package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tedwangl/go-util/pkg/cobrax"
	"github.com/tedwangl/go-util/pkg/daemon"
)

var (
	dbPath  = filepath.Join(os.Getenv("HOME"), ".devtool", "schedule.db")
	pidFile = filepath.Join(os.Getenv("HOME"), ".devtool", "schedule.pid")
)

// RegisterScheduleCommands 注册定时任务相关命令
func RegisterScheduleCommands(tool *cobrax.Tool) {
	scheduleGroup := cobrax.NewCommandGroup("schedule")

	// schedule start - 启动守护进程
	startCmd := tool.NewCommand(
		"start",
		"启动调度守护进程",
		"启动后台调度守护进程",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			// 检查是否已经运行
			if isRunning() {
				return fmt.Errorf("调度器已经在运行")
			}

			// 启动守护进程
			binary, err := os.Executable()
			if err != nil {
				return err
			}

			daemonCmd := exec.Command(binary, "daemon")
			daemonCmd.Stdout = nil
			daemonCmd.Stderr = nil
			daemonCmd.Stdin = nil
			daemonCmd.SysProcAttr = &syscall.SysProcAttr{
				Setsid: true,
			}

			if err := daemonCmd.Start(); err != nil {
				return fmt.Errorf("启动守护进程失败: %w", err)
			}

			// 保存 PID
			pidData := []byte(strconv.Itoa(daemonCmd.Process.Pid))
			if err := os.WriteFile(pidFile, pidData, 0644); err != nil {
				return fmt.Errorf("保存 PID 失败: %w", err)
			}

			fmt.Println("调度器已启动")
			return nil
		}),
	)

	// schedule stop - 停止守护进程
	stopCmd := tool.NewCommand(
		"stop",
		"停止调度守护进程",
		"停止后台调度守护进程",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if !isRunning() {
				return fmt.Errorf("调度器未运行")
			}

			pid, err := getPID()
			if err != nil {
				return err
			}

			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("查找进程失败: %w", err)
			}

			if err := process.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("停止进程失败: %w", err)
			}

			os.Remove(pidFile)
			fmt.Println("调度器已停止")
			return nil
		}),
	)

	// schedule status - 查看状态
	statusCmd := tool.NewCommand(
		"status",
		"查看调度器状态",
		"查看调度守护进程状态",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if isRunning() {
				pid, _ := getPID()
				fmt.Printf("调度器正在运行 (PID: %d)\n", pid)
			} else {
				fmt.Println("调度器未运行")
			}
			return nil
		}),
	)

	// schedule daemon - 守护进程（内部使用）
	daemonCmd := tool.NewCommand(
		"daemon",
		"守护进程（内部使用）",
		"后台守护进程，不要直接调用",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := d.Start(); err != nil {
				return err
			}

			fmt.Println("调度器守护进程已启动")

			// 监听信号
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)

			// 记录当前调度器中的任务（用于检测删除）
			currentTasks := make(map[string]bool)
			for _, job := range d.GetScheduler().ListJobs() {
				currentTasks[job.Name] = true
			}

			// 同步任务的函数（信号和定时器共用）
			syncTasks := func() {
				var tasks []daemon.Task
				if err := d.DB.Where("enabled = ? AND completed = ? AND schedule != ''", true, false).Find(&tasks).Error; err != nil {
					fmt.Printf("查询任务失败: %v\n", err)
					return
				}

				// 构建数据库任务集合
				dbTasks := make(map[string]*daemon.Task)
				for i := range tasks {
					dbTasks[tasks[i].Name] = &tasks[i]
				}

				// 1. 添加新任务（数据库有但调度器没有）
				for name, task := range dbTasks {
					if !currentTasks[name] {
						// 检查是否是特殊任务（@once 或 @delay）
						if task.Schedule == "@once" || (len(task.Schedule) > 7 && task.Schedule[:7] == "@delay:") {
							// 一次性任务或延迟任务：使用 goroutine 执行
							fmt.Printf("添加一次性/延迟任务: %s\n", name)
							go d.ExecuteOnceTask(task)
							currentTasks[name] = true
						} else {
							// 普通定时任务：添加到调度器
							fmt.Printf("添加定时任务: %s\n", name)
							if err := d.AddJobToScheduler(task); err != nil {
								fmt.Printf("添加失败: %v\n", err)
							} else {
								currentTasks[name] = true
								fmt.Printf("任务 %s 已添加到调度器\n", name)
							}
						}
					}
				}

				// 2. 删除任务（调度器有但数据库没有，或任务已完成）
				for taskName := range currentTasks {
					task, exists := dbTasks[taskName]
					if !exists || task.Completed {
						fmt.Printf("删除任务: %s\n", taskName)
						if err := d.RemoveJobFromScheduler(taskName); err != nil {
							fmt.Printf("删除失败: %v\n", err)
						} else {
							delete(currentTasks, taskName)
							fmt.Printf("任务 %s 已从调度器移除\n", taskName)
						}
					}
				}
			}

			// 定期同步定时器（每 30 秒检查一次，兜底机制）
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					// 定期同步
					fmt.Println("定期检查任务变化...")
					syncTasks()

				case sig := <-sigChan:
					switch sig {
					case syscall.SIGUSR1:
						// 添加任务信号：立即同步
						fmt.Println("收到添加任务信号，立即同步...")
						syncTasks()

					case syscall.SIGUSR2:
						// 删除任务信号：立即同步
						fmt.Println("收到删除任务信号，立即同步...")
						syncTasks()

					case syscall.SIGHUP:
						// 重载所有任务
						fmt.Println("收到重载信号，重新加载所有任务...")
						if err := d.Reload(); err != nil {
							fmt.Printf("重载失败: %v\n", err)
						} else {
							// 更新任务列表
							currentTasks = make(map[string]bool)
							for _, job := range d.GetScheduler().ListJobs() {
								currentTasks[job.Name] = true
							}
							fmt.Println("任务重载成功")
						}

					case syscall.SIGINT, syscall.SIGTERM:
						// 停止守护进程
						fmt.Println("\n收到停止信号，正在关闭...")
						return nil
					}
				}
			}
		}),
	)
	daemonCmd.Command.Hidden = true // 隐藏此命令

	// schedule list - 列出所有任务
	listCmd := tool.NewCommand(
		"list",
		"列出所有定时任务",
		"显示所有已配置的定时任务",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			tasks, err := d.ListTasks()
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("暂无定时任务")
				return nil
			}

			fmt.Println("定时任务列表:")
			fmt.Println("----------------------------------------")
			for i, task := range tasks {
				status := "运行中"
				if task.Completed {
					status = "已完成"
				} else if !task.Enabled {
					status = "已禁用"
				}
				scheduleInfo := "无调度"
				if task.Schedule != "" {
					scheduleInfo = task.Schedule
				}
				fmt.Printf("%d. [%s] %s (ID: %d)\n", i+1, status, task.Name, task.ID)
				fmt.Printf("   调度: %s\n", scheduleInfo)
				fmt.Printf("   命令: %s\n", task.Command)
				fmt.Printf("   创建: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
				if task.CompletedAt != nil {
					fmt.Printf("   完成: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
				}
				fmt.Println()
			}
			return nil
		}),
	)

	// schedule add - 添加任务
	addCmd := tool.NewCommand(
		"add",
		"添加定时任务",
		"添加新的定时任务、延迟任务或一次性任务",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("用法: devtool add <命令> [--schedule <cron> | --delay <时长> | --once]")
			}

			command := args[0]
			schedule := viper.GetString("schedule")
			delay := viper.GetString("delay")
			once := viper.GetBool("once")

			// 验证参数：必须指定 schedule、delay 或 once 之一
			if schedule == "" && delay == "" && !once {
				return fmt.Errorf("必须指定 --schedule、--delay 或 --once 之一")
			}
			if (schedule != "" && delay != "") || (schedule != "" && once) || (delay != "" && once) {
				return fmt.Errorf("--schedule、--delay 和 --once 只能指定一个")
			}

			name := viper.GetString("name")
			if name == "" {
				name = fmt.Sprintf("task-%d", time.Now().Unix())
			}

			// 构建 schedule 字符串
			var scheduleStr string
			var runAt *time.Time
			if once {
				scheduleStr = "@once"
				now := time.Now()
				runAt = &now
			} else if delay != "" {
				duration, err := time.ParseDuration(delay)
				if err != nil {
					return fmt.Errorf("无效的延迟时间格式: %v（示例: 5m, 1h, 30s）", err)
				}
				scheduleStr = "@delay:" + delay
				runAtTime := time.Now().Add(duration)
				runAt = &runAtTime
			} else {
				scheduleStr = schedule
			}

			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := d.AddTaskWithRunAt(name, command, scheduleStr, runAt); err != nil {
				return err
			}

			fmt.Printf("任务 %s 添加成功\n", name)
			if once {
				fmt.Printf("类型: 一次性任务（立即执行）\n")
			} else if delay != "" {
				fmt.Printf("类型: 延迟任务（%s 后执行）\n", delay)
				fmt.Printf("执行时间: %s\n", runAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("调度: %s\n", schedule)
			}
			fmt.Printf("命令: %s\n", command)

			// 通知守护进程添加任务
			if isRunning() {
				pid, _ := getPID()
				process, err := os.FindProcess(pid)
				if err == nil {
					process.Signal(syscall.SIGUSR1)
					fmt.Println("已通知守护进程添加任务")
				}
			} else {
				fmt.Println("\n提示: 使用 'devtool start' 启动调度器")
			}

			return nil
		}),
	)
	addCmd.AddFlag("name", "n", "", "任务名称")
	addCmd.AddFlag("schedule", "s", "", "cron 表达式（定时任务）")
	addCmd.AddFlag("delay", "", "", "延迟时间（如: 5m, 1h, 30s）")
	addCmd.AddFlag("once", "o", false, "立即执行一次")

	// schedule remove - 删除任务
	removeCmd := tool.NewCommand(
		"remove",
		"删除定时任务",
		"删除指定的定时任务",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要删除的任务名称")
			}

			name := args[0]
			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := d.RemoveTask(name); err != nil {
				return err
			}

			fmt.Printf("任务 %s 已从数据库删除\n", name)

			// 通知守护进程移除任务
			if isRunning() {
				pid, _ := getPID()
				process, err := os.FindProcess(pid)
				if err == nil {
					process.Signal(syscall.SIGUSR2)
					fmt.Println("已通知守护进程移除任务")
				}
			}

			return nil
		}),
	)

	// schedule logs - 查看日志（只显示状态）
	logsCmd := tool.NewCommand(
		"logs",
		"查看任务执行日志",
		"查看任务执行历史记录（只显示状态）",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			taskName := ""
			if len(args) > 0 {
				taskName = args[0]
			}

			limit := viper.GetInt("limit")
			if limit == 0 {
				limit = 20
			}

			daemon, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer daemon.Close()

			logs, err := daemon.ListLogs(taskName, limit)
			if err != nil {
				return err
			}

			if len(logs) == 0 {
				fmt.Println("暂无执行日志")
				return nil
			}

			fmt.Println("任务执行日志:")
			fmt.Println("----------------------------------------")
			for _, log := range logs {
				duration := ""
				if log.EndTime != nil {
					duration = fmt.Sprintf(" (耗时: %s)", log.EndTime.Sub(log.StartTime).Round(time.Millisecond))
				}

				fmt.Printf("[%s] %s - %s%s\n",
					log.StartTime.Format("2006-01-02 15:04:05"),
					log.TaskName,
					log.Status,
					duration,
				)
			}
			return nil
		}),
	)
	logsCmd.AddFlag("limit", "l", 20, "显示条数")

	// schedule clean - 清理已完成任务
	cleanCmd := tool.NewCommand(
		"clean",
		"清理已完成任务",
		"删除所有已完成的一次性/延迟任务记录",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			// 删除已完成的任务
			result := d.DB.Where("completed = ?", true).Delete(&daemon.Task{})
			if result.Error != nil {
				return result.Error
			}

			fmt.Printf("已清理 %d 个已完成任务\n", result.RowsAffected)
			return nil
		}),
	)

	scheduleGroup.AddCommand(startCmd, stopCmd, statusCmd, listCmd, addCmd, removeCmd, logsCmd, cleanCmd, daemonCmd)
	tool.AddGroupLogic(scheduleGroup)
}

// isRunning 检查守护进程是否运行
func isRunning() bool {
	pid, err := getPID()
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 发送信号 0 检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// getPID 获取守护进程 PID
func getPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}

	return pid, nil
}
