package cobrax

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewTool 创建一个新的命令行工具
func NewTool(name, version, desc string) *Tool {
	rootCmd := &Command{
		Command: &cobra.Command{
			Use:   name,
			Short: desc,
			Long:  desc,
		},
		ErrHandler: DefaultErrorHandler,
	}

	tool := &Tool{
		rootCmd:    rootCmd,
		name:       name,
		version:    version,
		desc:       desc,
		errHandler: DefaultErrorHandler,
		envPrefix:  "CLI", // 默认环境变量前缀
	}

	tool.AddVersionCommand()
	tool.AddTreeCommand()
	tool.SetGlobalFlags()
	return tool
}

// SetErrorHandler 设置全局错误处理函数
func (t *Tool) SetErrorHandler(handler ErrorHandler) {
	if handler != nil {
		t.errHandler = handler
		t.rootCmd.ErrHandler = handler
	}
}

// GetRootCommand 获取根命令
func (t *Tool) GetRootCommand() *Command {
	return t.rootCmd
}

// AddVersionCommand 添加版本命令
func (t *Tool) AddVersionCommand() {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示工具版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s\n", t.name, t.version)
		},
	}
	t.rootCmd.Command.AddCommand(versionCmd)
}

// AddTreeCommand 添加树形结构命令
func (t *Tool) AddTreeCommand() {
	treeCmd := &cobra.Command{
		Use:   "tree",
		Short: "显示命令树形结构",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(t.PrintCommandTree())
		},
	}
	t.rootCmd.Command.AddCommand(treeCmd)
}

// SetGlobalFlags 设置全局标志
func (t *Tool) SetGlobalFlags() {
	t.rootCmd.PersistentFlags().BoolP("verbose", "v", false, "显示详细信息")
	t.rootCmd.PersistentFlags().BoolP("debug", "d", false, "显示调试信息")
	t.rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")
}

// Execute 执行命令
func (t *Tool) Execute() int {
	if t.errHandler != nil {
		t.rootCmd.ErrHandler = t.errHandler
	}

	// 捕获panic
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("程序崩溃: %v\n堆栈跟踪:\n%s\n", r, debug.Stack())
				if t.logger != nil {
					t.logger.Fatal("程序崩溃", zap.Any("panic", r), zap.Stack("stack"))
				}
				fmt.Fprint(os.Stderr, errMsg)
				close(done)
			}
		}()

		if err := t.rootCmd.Command.Execute(); err != nil {
			if handler := t.errHandler; handler != nil {
				handler(err, t.rootCmd.Command)
			}
		}
		close(done)
	}()

	<-done
	return 0
}

// NewCommand 创建一个新的子命令
func (t *Tool) NewCommand(use, short, long string, runner CmdRunner, subCmds ...*Command) *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   use,
			Short: short,
			Long:  long,
		},
		Runner:     runner,
		ErrHandler: t.errHandler,
		validators: make(map[string][]ParamValidator),
	}

	cmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		// 执行参数校验
		if err := cmd.ValidateFlags(); err != nil {
			if t.logger != nil {
				t.logger.Warn("参数校验失败",
					zap.String("command", cobraCmd.CommandPath()),
					zap.Error(err),
				)
			}
			return err
		}

		// 执行命令
		if cmd.Runner != nil {
			if t.logger != nil {
				t.logger.Info("执行命令", zap.String("command", cobraCmd.CommandPath()))
			}
			return cmd.Runner.Run(cobraCmd, args)
		}
		return nil
	}

	if len(subCmds) > 0 {
		for _, subCmd := range subCmds {
			cmd.AddCommand(subCmd)
		}
	}

	return cmd
}

// AddCommand 添加命令到工具
func (t *Tool) AddCommand(cmds ...*Command) {
	if len(cmds) > 0 {
		for _, cmd := range cmds {
			t.rootCmd.Command.AddCommand(cmd.Command)
		}
	}
}

// AddGroupLogic 添加逻辑分组（仅用于帮助信息分类）
func (t *Tool) AddGroupLogic(cmdGroup *CommandGroup) {
	group := &cobra.Group{
		ID:    cmdGroup.Name,
		Title: fmt.Sprintf("%s Commands", strings.ToUpper(cmdGroup.Name[:1])+cmdGroup.Name[1:]),
	}
	t.rootCmd.Command.AddGroup(group)

	for _, cmd := range cmdGroup.Commands {
		cmd.Command.GroupID = group.ID
		t.rootCmd.Command.AddCommand(cmd.Command)
	}
}

// AddGroupNested 添加真实嵌套分组（父子命令关系，可传递 PersistentFlags）
// 使用示例：
//
//	dbCmd := tool.NewCommand("db", "数据库操作", "数据库相关命令", nil)
//	dbCmd.AddPersistentFlag("host", "h", "localhost", "数据库主机")
//	tool.AddGroupNested(dbCmd, migrateCmd, backupCmd)
//	// 调用：mycli db migrate --host=192.168.1.1
func (t *Tool) AddGroupNested(parentCmd *Command, subCmds ...*Command) {
	for _, subCmd := range subCmds {
		parentCmd.AddCommand(subCmd)
	}
	t.rootCmd.Command.AddCommand(parentCmd.Command)
}

// NewLogger 创建zap日志器
func NewLogger(cfg LoggerConfig) (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	var outputs []string
	if cfg.Console {
		outputs = append(outputs, "stdout")
	}
	if cfg.File && cfg.Path != "" {
		// 自动创建日志目录
		if err := ensureDir(cfg.Path); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}
		outputs = append(outputs, cfg.Path)
	}

	if len(outputs) == 0 {
		// 不输出日志，使用 nop logger
		return zap.NewNop(), nil
	}

	config.OutputPaths = outputs
	return config.Build()
}

// ensureDir 确保文件所在目录存在
func ensureDir(filePath string) error {
	dir := filePath[:len(filePath)-len(filePath[len(filePath)-1:])]
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '/' {
			dir = filePath[:i]
			break
		}
	}
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

// InitDefaultLogger 初始化默认日志器并设置到 Tool
func (t *Tool) InitDefaultLogger(cfg LoggerConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("初始化日志器失败: %w", err)
	}
	t.logger = logger
	return nil
}

// IsVerbose 获取 verbose 标志值
func (t *Tool) IsVerbose() bool {
	verbose, _ := t.rootCmd.PersistentFlags().GetBool("verbose")
	return verbose
}

// IsDebug 获取 debug 标志值
func (t *Tool) IsDebug() bool {
	debug, _ := t.rootCmd.PersistentFlags().GetBool("debug")
	return debug
}

// GetConfigPath 获取配置文件路径
func (t *Tool) GetConfigPath() string {
	path, _ := t.rootCmd.PersistentFlags().GetString("config")
	return path
}

// Info 记录 Info 级别日志
func (t *Tool) Info(msg string, fields ...zap.Field) {
	if t.logger != nil {
		t.logger.Info(msg, fields...)
	}
}

// Warn 记录 Warn 级别日志
func (t *Tool) Warn(msg string, fields ...zap.Field) {
	if t.logger != nil {
		t.logger.Warn(msg, fields...)
	}
}

// Error 记录 Error 级别日志
func (t *Tool) Error(msg string, fields ...zap.Field) {
	if t.logger != nil {
		t.logger.Error(msg, fields...)
	}
}

// Debug 记录 Debug 级别日志
func (t *Tool) Debug(msg string, fields ...zap.Field) {
	if t.logger != nil {
		t.logger.Debug(msg, fields...)
	}
}

// Fatal 记录 Fatal 级别日志并退出
func (t *Tool) Fatal(msg string, fields ...zap.Field) {
	if t.logger != nil {
		t.logger.Fatal(msg, fields...)
	}
}

// GetLogger 获取日志器
func (t *Tool) GetLogger() *zap.Logger {
	return t.logger
}

// SetEnvPrefix 设置环境变量前缀（默认为 "CLI"）
func (t *Tool) SetEnvPrefix(prefix string) {
	t.envPrefix = prefix
}

// GetEnvPrefix 获取环境变量前缀
func (t *Tool) GetEnvPrefix() string {
	return t.envPrefix
}
