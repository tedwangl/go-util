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
	dbPath     = filepath.Join(os.Getenv("HOME"), ".devtool", "schedule.db")
	pidFile    = filepath.Join(os.Getenv("HOME"), ".devtool", "schedule.pid")
	actionFile = filepath.Join(os.Getenv("HOME"), ".devtool", "schedule.action")
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
			if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", daemonCmd.Process.Pid)), 0644); err != nil {
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
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

			for {
				sig := <-sigChan
				switch sig {
				case syscall.SIGHUP:
					// 读取操作文件
					data, err := os.ReadFile(actionFile)
					if err != nil {
						fmt.Printf("读取操作文件失败: %v\n", err)
						continue
					}

					// 解析操作：remove:taskname 或 reload
					action := string(data)
					if action == "reload" {
						fmt.Println("收到重载信号，重新加载任务...")
						if err := d.Reload(); err != nil {
							fmt.Printf("重载失败: %v\n", err)
						} else {
							fmt.Println("任务重载成功")
						}
					} else if len(action) > 7 && action[:7] == "remove:" {
						taskName := action[7:]
						fmt.Printf("收到移除信号，移除任务: %s\n", taskName)
						if err := d.RemoveJobFromScheduler(taskName); err != nil {
							fmt.Printf("移除失败: %v\n", err)
						} else {
							fmt.Printf("任务 %s 已从调度器移除\n", taskName)
						}
					}

					// 删除操作文件
					os.Remove(actionFile)

				case syscall.SIGINT, syscall.SIGTERM:
					// 停止守护进程
					fmt.Println("\n收到停止信号，正在关闭...")
					return nil
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
				status := "启用"
				if !task.Enabled {
					status = "禁用"
				}
				scheduleInfo := "无调度"
				if task.Schedule != "" {
					scheduleInfo = task.Schedule
				}
				fmt.Printf("%d. [%s] %s (ID: %d)\n", i+1, status, task.Name, task.ID)
				fmt.Printf("   调度: %s\n", scheduleInfo)
				fmt.Printf("   命令: %s\n", task.Command)
				fmt.Printf("   创建: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Println()
			}
			return nil
		}),
	)

	// schedule add - 添加任务
	addCmd := tool.NewCommand(
		"add",
		"添加定时任务",
		"添加新的定时任务",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("用法: devtool add <命令> --schedule <cron表达式> [--name 任务名]")
			}

			command := args[0]
			schedule := viper.GetString("schedule")
			if schedule == "" {
				return fmt.Errorf("必须指定调度表达式（--schedule）")
			}

			name := viper.GetString("name")
			if name == "" {
				name = fmt.Sprintf("task-%d", time.Now().Unix())
			}

			d, err := daemon.NewDaemon(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := d.AddTask(name, command, schedule); err != nil {
				return err
			}

			fmt.Printf("任务 %s 添加成功\n", name)
			fmt.Printf("调度: %s\n", schedule)
			fmt.Printf("命令: %s\n", command)

			// 通知守护进程重新加载
			if isRunning() {
				// 写入操作文件
				if err := os.WriteFile(actionFile, []byte("reload"), 0644); err != nil {
					fmt.Printf("写入操作文件失败: %v\n", err)
				} else {
					// 发送信号
					pid, _ := getPID()
					process, err := os.FindProcess(pid)
					if err == nil {
						process.Signal(syscall.SIGHUP)
						fmt.Println("已通知守护进程重新加载")
					}
				}
			} else {
				fmt.Println("\n提示: 使用 'devtool start' 启动调度器")
			}

			return nil
		}),
	)
	addCmd.AddFlag("name", "n", "", "任务名称")
	addCmd.AddFlag("schedule", "s", "", "cron 表达式（必填）")

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
				// 写入操作文件
				if err := os.WriteFile(actionFile, []byte("remove:"+name), 0644); err != nil {
					fmt.Printf("写入操作文件失败: %v\n", err)
				} else {
					// 发送信号
					pid, _ := getPID()
					process, err := os.FindProcess(pid)
					if err == nil {
						process.Signal(syscall.SIGHUP)
						fmt.Println("已通知守护进程移除任务")
					}
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

	scheduleGroup.AddCommand(startCmd, stopCmd, statusCmd, listCmd, addCmd, removeCmd, logsCmd, daemonCmd)
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
