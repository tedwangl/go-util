package main

import (
	"fmt"
	"os"

	"github.com/tedwangl/go-util/cmd/devtool/commands"
	"github.com/tedwangl/go-util/pkg/cobrax"
)

func main() {
	// 创建工具
	tool := cobrax.NewTool("devtool", "1.0.0", "个人开发工具集")

	// 设置环境变量前缀
	tool.SetEnvPrefix("DEVTOOL")

	// 初始化日志器
	if err := tool.InitDefaultLogger(cobrax.LoggerConfig{
		Console: true,
		File:    true,
		Path:    os.ExpandEnv("$HOME/.devtool/logs/app.log"),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志器失败: %v\n", err)
		os.Exit(1)
	}

	// 设置配置文件
	tool.SetConfig(os.ExpandEnv("$HOME/.devtool/config.yaml"))

	// 设置错误处理器
	tool.SetErrorHandler(cobrax.LoggingErrorHandler(tool.GetLogger()))

	// 注册命令组
	commands.RegisterPyCommands(tool)
	commands.RegisterScheduleCommands(tool)
	commands.RegisterNetCommands(tool)
	commands.RegisterGoCommands(tool)

	// 执行
	os.Exit(tool.Execute())
}
