package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tedwangl/go-util/pkg/cobrax"
)

// RegisterGoCommands 注册 Go 相关命令
func RegisterGoCommands(tool *cobrax.Tool) {
	goGroup := cobrax.NewCommandGroup("go")

	// go test - 运行测试
	testCmd := tool.NewCommand(
		"test",
		"运行 Go 测试",
		"运行 Go 测试，支持常用参数",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			cmdArgs := []string{"test"}

			// 添加参数
			if viper.GetBool("verbose") {
				cmdArgs = append(cmdArgs, "-v")
			}
			if viper.GetBool("cover") {
				cmdArgs = append(cmdArgs, "-cover")
			}
			if viper.GetBool("race") {
				cmdArgs = append(cmdArgs, "-race")
			}
			if viper.GetInt("count") > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("-count=%d", viper.GetInt("count")))
			}
			if viper.GetString("run") != "" {
				cmdArgs = append(cmdArgs, fmt.Sprintf("-run=%s", viper.GetString("run")))
			}

			// 添加包路径
			if len(args) > 0 {
				cmdArgs = append(cmdArgs, args...)
			} else {
				cmdArgs = append(cmdArgs, "./...")
			}

			goCmd := exec.Command("go", cmdArgs...)
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)
	testCmd.AddFlag("cover", "", false, "显示覆盖率")
	testCmd.AddFlag("race", "r", false, "启用竞态检测")
	testCmd.AddFlag("count", "", 1, "运行次数")
	testCmd.AddFlag("run", "", "", "运行匹配的测试")

	// go bench - 运行基准测试
	benchCmd := tool.NewCommand(
		"bench",
		"运行基准测试",
		"运行 Go 基准测试",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			cmdArgs := []string{"test", "-bench=."}

			if viper.GetInt("benchtime") > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("-benchtime=%ds", viper.GetInt("benchtime")))
			}
			if viper.GetBool("benchmem") {
				cmdArgs = append(cmdArgs, "-benchmem")
			}

			if len(args) > 0 {
				cmdArgs = append(cmdArgs, args...)
			} else {
				cmdArgs = append(cmdArgs, "./...")
			}

			goCmd := exec.Command("go", cmdArgs...)
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)
	benchCmd.AddFlag("benchtime", "t", 1, "基准测试时间（秒）")
	benchCmd.AddFlag("benchmem", "m", false, "显示内存分配")

	// go get - 下载包
	getCmd := tool.NewCommand(
		"get",
		"下载 Go 包",
		"下载并安装 Go 包",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要下载的包")
			}

			cmdArgs := []string{"get"}
			if viper.GetBool("update") {
				cmdArgs = append(cmdArgs, "-u")
			}
			cmdArgs = append(cmdArgs, args...)

			goCmd := exec.Command("go", cmdArgs...)
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)
	getCmd.AddFlag("update", "u", false, "更新到最新版本")

	// go mod - 模块管理
	modCmd := tool.NewCommand(
		"mod",
		"Go 模块管理",
		"Go 模块管理命令",
		nil,
	)
	modCmd.Command.GroupID = "go"

	// go mod tidy
	tidyCmd := tool.NewCommand(
		"tidy",
		"整理依赖",
		"添加缺失的依赖，删除未使用的依赖",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			goCmd := exec.Command("go", "mod", "tidy")
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)

	// go mod download
	downloadCmd := tool.NewCommand(
		"download",
		"下载依赖",
		"下载所有依赖到本地缓存",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			goCmd := exec.Command("go", "mod", "download")
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)

	// go mod vendor
	vendorCmd := tool.NewCommand(
		"vendor",
		"创建 vendor 目录",
		"将依赖复制到 vendor 目录",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			goCmd := exec.Command("go", "mod", "vendor")
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)

	modCmd.Command.AddCommand(tidyCmd.Command, downloadCmd.Command, vendorCmd.Command)

	// go build - 构建
	buildCmd := tool.NewCommand(
		"build",
		"构建 Go 程序",
		"编译 Go 程序",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			cmdArgs := []string{"build"}

			if viper.GetString("output") != "" {
				cmdArgs = append(cmdArgs, "-o", viper.GetString("output"))
			}
			if viper.GetBool("race") {
				cmdArgs = append(cmdArgs, "-race")
			}

			cmdArgs = append(cmdArgs, args...)

			goCmd := exec.Command("go", cmdArgs...)
			goCmd.Stdout = os.Stdout
			goCmd.Stderr = os.Stderr
			return goCmd.Run()
		}),
	)
	buildCmd.AddFlag("output", "o", "", "输出文件名")
	buildCmd.AddFlag("race", "r", false, "启用竞态检测")

	goGroup.AddCommand(testCmd, benchCmd, getCmd, modCmd, buildCmd)
	tool.AddGroupLogic(goGroup)
}
