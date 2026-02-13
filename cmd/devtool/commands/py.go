package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tedwangl/go-util/pkg/cobrax"
	"github.com/tedwangl/go-util/pkg/conda"
)

// RegisterPyCommands 注册 Python/Conda 相关命令
func RegisterPyCommands(tool *cobrax.Tool) {
	pyGroup := cobrax.NewCommandGroup("python")

	// conda envs - 列出所有环境
	envsCmd := tool.NewCommand(
		"envs",
		"列出所有 conda 环境",
		"显示所有 conda 环境（* 表示当前环境）",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("conda", "env", "list")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// conda activate - 切换环境（提示用户）
	activateCmd := tool.NewCommand(
		"activate",
		"切换 conda 环境",
		"切换到指定的 conda 环境（显示激活命令）",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定环境名称")
			}

			envName := args[0]
			fmt.Printf("请在终端执行以下命令切换环境:\n")
			fmt.Printf("  conda activate %s\n", envName)
			return nil
		}),
	)

	// conda remove-env - 删除环境
	removeEnvCmd := tool.NewCommand(
		"remove-env",
		"删除 conda 环境",
		"删除指定的 conda 环境",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要删除的环境名称")
			}

			envName := args[0]
			c := exec.Command("conda", "env", "remove", "-n", envName, "-y")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// conda install - 安装包
	installCmd := tool.NewCommand(
		"install",
		"安装 conda 包",
		"在当前环境安装 conda 包",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要安装的包名")
			}

			cmdArgs := append([]string{"install", "-y"}, args...)
			c := exec.Command("conda", cmdArgs...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// conda channels - 查看镜像源
	channelsCmd := tool.NewCommand(
		"channels",
		"查看镜像源",
		"显示当前配置的 conda 镜像源",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			c := exec.Command("conda", "config", "--show", "channels")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// conda add-channel - 添加镜像源
	addChannelCmd := tool.NewCommand(
		"add-channel",
		"添加镜像源",
		"添加 conda 镜像源地址",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定镜像源地址")
			}

			channel := args[0]
			c := exec.Command("conda", "config", "--add", "channels", channel)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// conda remove-channel - 删除镜像源
	removeChannelCmd := tool.NewCommand(
		"remove-channel",
		"删除镜像源",
		"删除指定的 conda 镜像源",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要删除的镜像源地址")
			}

			channel := args[0]
			c := exec.Command("conda", "config", "--remove", "channels", channel)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		}),
	)

	// py run - 运行 Python 脚本
	runCmd := tool.NewCommand(
		"run",
		"运行 Python 脚本",
		"在指定 conda 环境运行 Python 脚本",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要运行的 Python 脚本")
			}

			envName := viper.GetString("env")
			if envName == "" {
				envName = conda.GetCurrentEnv()
			}

			script := args[0]
			scriptArgs := args[1:]

			fmt.Printf("在环境 %s 中运行: %s\n", envName, script)
			return conda.RunPython(envName, script, scriptArgs...)
		}),
	)
	runCmd.AddFlag("env", "e", "", "环境名称（默认当前环境）")

	// py exec - 执行 Python 命令
	execCmd := tool.NewCommand(
		"exec",
		"执行 Python 命令",
		"在指定环境执行 Python 命令",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要执行的 Python 命令")
			}

			envName := viper.GetString("env")
			if envName == "" {
				envName = conda.GetCurrentEnv()
			}

			command := args[0]
			output, err := conda.RunPythonCommand(envName, command)
			if err != nil {
				return err
			}

			fmt.Println(output)
			return nil
		}),
	)
	execCmd.AddFlag("env", "e", "", "环境名称（默认当前环境）")

	// py pip - pip 包管理
	pipCmd := tool.NewCommand(
		"pip",
		"pip 包管理",
		"使用 pip 管理 Python 包",
		nil,
	)
	pipCmd.Command.GroupID = "python"

	// py pip install
	pipInstallCmd := tool.NewCommand(
		"install",
		"安装 pip 包",
		"使用 pip 安装 Python 包",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要安装的包名")
			}

			envName := viper.GetString("env")
			if envName == "" {
				envName = conda.GetCurrentEnv()
			}

			packageName := args[0]
			fmt.Printf("在环境 %s 中使用 pip 安装 %s...\n", envName, packageName)
			return conda.PipInstall(envName, packageName)
		}),
	)
	pipInstallCmd.AddFlag("env", "e", "", "环境名称（默认当前环境）")

	// py pip uninstall
	pipUninstallCmd := tool.NewCommand(
		"uninstall",
		"卸载 pip 包",
		"使用 pip 卸载 Python 包",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要卸载的包名")
			}

			envName := viper.GetString("env")
			if envName == "" {
				envName = conda.GetCurrentEnv()
			}

			packageName := args[0]
			fmt.Printf("在环境 %s 中使用 pip 卸载 %s...\n", envName, packageName)
			return conda.PipUninstall(envName, packageName)
		}),
	)
	pipUninstallCmd.AddFlag("env", "e", "", "环境名称（默认当前环境）")

	// py pip list
	pipListCmd := tool.NewCommand(
		"list",
		"列出 pip 包",
		"列出 pip 安装的所有包",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			envName := viper.GetString("env")
			if envName == "" {
				envName = conda.GetCurrentEnv()
			}

			packages, err := conda.PipList(envName)
			if err != nil {
				return err
			}

			fmt.Printf("环境 %s 的 pip 包列表:\n", envName)
			fmt.Println("----------------------------------------")
			for _, pkg := range packages {
				fmt.Printf("%-40s %s\n", pkg.Name, pkg.Version)
			}
			return nil
		}),
	)
	pipListCmd.AddFlag("env", "e", "", "环境名称（默认当前环境）")

	pipCmd.Command.AddCommand(pipInstallCmd.Command, pipUninstallCmd.Command, pipListCmd.Command)
	pyGroup.AddCommand(envsCmd, activateCmd, removeEnvCmd, installCmd, channelsCmd, addChannelCmd, removeChannelCmd, runCmd, execCmd, pipCmd)
	tool.AddGroupLogic(pyGroup)
}
