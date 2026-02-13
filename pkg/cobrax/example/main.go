package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tedwangl/go-util/pkg/cobrax"
	"go.uber.org/zap"
)

func main() {
	// 创建工具
	tool := cobrax.NewTool("mycli", "1.0.0", "一个完整的 CLI 工具示例")

	// 设置环境变量前缀（可选，默认为 "CLI"）
	tool.SetEnvPrefix("MYCLI")

	// 初始化日志器（自动创建日志目录）
	if err := tool.InitDefaultLogger(cobrax.LoggerConfig{
		Console: true,
		File:    true,
		Path:    "./logs/app.log",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志器失败: %v\n", err)
		os.Exit(1)
	}

	// 设置配置文件（可选）
	tool.SetConfig("")

	// 设置错误处理器
	tool.SetErrorHandler(cobrax.LoggingErrorHandler(tool.GetLogger()))

	// ==================== 独立命令 ====================
	helloCmd := tool.NewCommand(
		"hello",
		"打招呼命令",
		"向指定用户打招呼",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			name := viper.GetString("name")
			age := viper.GetInt("age")

			tool.Info("执行 hello 命令",
				zap.String("name", name),
				zap.Int("age", age),
			)

			fmt.Printf("Hello, %s! You are %d years old.\n", name, age)
			return nil
		}),
	)

	// 添加标志和校验器
	helloCmd.AddFlag("name", "n", "", "用户名")
	helloCmd.AddFlag("age", "a", 0, "年龄")

	helloCmd.AddParamValidator("name", &cobrax.RequiredValidator{Message: "用户名不能为空"})
	helloCmd.AddParamValidator("name", &cobrax.MinLengthValidator{Min: 2, Message: "用户名至少2个字符"})
	helloCmd.AddParamValidator("age", &cobrax.MinValueValidator{Min: 1, Message: "年龄必须大于0"})
	helloCmd.AddParamValidator("age", &cobrax.MaxValueValidator{Max: 150, Message: "年龄不能超过150"})

	tool.AddCommand(helloCmd)

	// ==================== 逻辑分组命令（仅帮助信息分类）====================
	userGroup := cobrax.NewCommandGroup("user")

	// user list 命令
	listCmd := tool.NewCommand(
		"list",
		"列出用户",
		"列出所有用户信息",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			limit := viper.GetInt("limit")
			offset := viper.GetInt("offset")

			tool.Info("执行 list 命令",
				zap.Int("limit", limit),
				zap.Int("offset", offset),
			)

			fmt.Printf("Listing users (limit=%d, offset=%d)\n", limit, offset)
			fmt.Println("1. Alice")
			fmt.Println("2. Bob")
			fmt.Println("3. Charlie")
			return nil
		}),
	)
	listCmd.AddFlag("limit", "l", 10, "每页数量")
	listCmd.AddFlag("offset", "o", 0, "偏移量")
	listCmd.AddParamValidator("limit", &cobrax.MinValueValidator{Min: 1})
	listCmd.AddParamValidator("limit", &cobrax.MaxValueValidator{Max: 100})

	// user create 命令
	createCmd := tool.NewCommand(
		"create",
		"创建用户",
		"创建一个新用户",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			username := viper.GetString("username")
			email := viper.GetString("email")

			tool.Info("执行 create 命令",
				zap.String("username", username),
				zap.String("email", email),
			)

			fmt.Printf("User created: %s (%s)\n", username, email)
			return nil
		}),
	)
	createCmd.AddFlag("username", "u", "", "用户名")
	createCmd.AddFlag("email", "e", "", "邮箱")
	createCmd.AddParamValidator("username", &cobrax.RequiredValidator{})
	createCmd.AddParamValidator("username", &cobrax.MinLengthValidator{Min: 3})
	createCmd.AddParamValidator("email", &cobrax.RequiredValidator{})
	createCmd.AddParamValidator("email", &cobrax.RegexValidator{
		Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		Message: "邮箱格式不正确",
	})

	userGroup.AddCommand(listCmd, createCmd)
	tool.AddGroupLogic(userGroup)

	// ==================== 真实嵌套分组（父子命令，可传递 PersistentFlags）====================
	// db 父命令
	dbCmd := tool.NewCommand(
		"db",
		"数据库操作",
		"数据库相关命令",
		nil, // 父命令不需要 Runner
	)

	// 添加持久化标志（子命令会继承）
	dbCmd.AddPersistentFlag("host", "H", "localhost", "数据库主机")
	dbCmd.AddPersistentFlag("port", "P", 3306, "数据库端口")
	dbCmd.AddPersistentFlag("database", "D", "mydb", "数据库名")

	// 使用 ChainPersistentPreRunE 自动链式调用父钩子
	dbCmd.ChainPersistentPreRunE(tool.GetRootCommand(), func(cmd *cobra.Command, args []string) error {
		host := viper.GetString("host")
		port := viper.GetInt("port")
		database := viper.GetString("database")

		tool.Info("数据库连接信息",
			zap.String("host", host),
			zap.Int("port", port),
			zap.String("database", database),
		)

		fmt.Printf("Connecting to %s:%d/%s...\n", host, port, database)
		return nil
	})

	// 使用 ChainPersistentPostRunE 自动链式调用父钩子
	dbCmd.ChainPersistentPostRunE(tool.GetRootCommand(), func(cmd *cobra.Command, args []string) error {
		tool.Info("关闭数据库连接")
		fmt.Println("Connection closed.")
		return nil
	})

	// db migrate 子命令
	migrateCmd := tool.NewCommand(
		"migrate",
		"执行数据库迁移",
		"执行数据库迁移脚本",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			version := viper.GetString("version")

			tool.Info("执行迁移", zap.String("version", version))

			fmt.Printf("Running migration to version: %s\n", version)
			return nil
		}),
	)
	migrateCmd.AddFlag("version", "", "latest", "迁移版本")

	// db backup 子命令
	backupCmd := tool.NewCommand(
		"backup",
		"备份数据库",
		"创建数据库备份",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			output := viper.GetString("output")

			tool.Info("执行备份", zap.String("output", output))

			fmt.Printf("Backing up database to: %s\n", output)
			return nil
		}),
	)
	backupCmd.AddFlag("output", "o", "./backup.sql", "备份文件路径")
	backupCmd.AddParamValidator("output", &cobrax.RequiredValidator{})

	// 添加嵌套分组
	tool.AddGroupNested(dbCmd, migrateCmd, backupCmd)

	// ==================== 配置测试命令 ====================
	configCmd := tool.NewCommand(
		"config",
		"显示配置",
		"显示当前配置信息",
		cobrax.CmdRunnerFunc(func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== 配置信息 ===")
			fmt.Printf("verbose: %v\n", tool.IsVerbose())
			fmt.Printf("debug: %v\n", tool.IsDebug())
			fmt.Printf("config file: %s\n", tool.GetConfigPath())

			// 直接用 viper 读取（命令行 > 环境变量 > 配置文件）
			fmt.Printf("\napp-name: %s\n", viper.GetString("app-name"))
			fmt.Printf("app-port: %d\n", viper.GetInt("app-port"))
			fmt.Printf("app-debug: %v\n", viper.GetBool("app-debug"))

			return nil
		}),
	)
	configCmd.AddFlag("app-name", "", "", "应用名称")
	configCmd.AddFlag("app-port", "", 0, "应用端口")
	configCmd.AddFlag("app-debug", "", false, "调试模式")
	tool.AddCommand(configCmd)

	// 执行
	os.Exit(tool.Execute())
}
