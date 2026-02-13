package cobrax

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type (
	// ==================== 接口定义 ====================

	// CmdRunner 定义命令运行器接口
	CmdRunner interface {
		Run(cmd *cobra.Command, args []string) error
	}

	// ParamValidator 定义参数校验器接口
	ParamValidator interface {
		Validate(value any) error
	}

	// ErrorHandler 定义错误处理函数类型
	ErrorHandler func(err error, cmd *cobra.Command) error

	// ==================== 核心类型 ====================

	// Tool 表示一个命令行工具，管理全局配置和命令集
	Tool struct {
		rootCmd    *Command
		name       string
		version    string
		desc       string
		errHandler ErrorHandler
		logger     *zap.Logger
		envPrefix  string // 环境变量前缀
	}

	// Command 是对cobra.Command的包装，提供更简洁的API
	Command struct {
		*cobra.Command
		Runner     CmdRunner
		ErrHandler ErrorHandler
		validators map[string][]ParamValidator
	}

	// ==================== 辅助类型 ====================

	// CmdRunnerFunc 是函数类型的CmdRunner实现
	CmdRunnerFunc func(cmd *cobra.Command, args []string) error

	// Flag 标志定义
	Flag struct {
		Name         string
		Shorthand    string
		DefaultValue any
		Usage        string
	}

	// CommandGroup 命令组
	CommandGroup struct {
		Name     string
		Commands []*Command
	}

	// LoggerConfig 日志配置
	LoggerConfig struct {
		Console bool   // 是否输出到控制台
		File    bool   // 是否输出到文件
		Path    string // 文件路径（File=true 时必填）
	}

	// ==================== 校验器类型 ====================

	// RequiredValidator 检查参数是否必填
	RequiredValidator struct {
		Message string
	}

	// MinLengthValidator 检查字符串最小长度
	MinLengthValidator struct {
		Min     int
		Message string
	}

	// MaxLengthValidator 检查字符串最大长度
	MaxLengthValidator struct {
		Max     int
		Message string
	}

	// RegexValidator 使用正则表达式验证字符串
	RegexValidator struct {
		Pattern string
		Message string
	}

	// MinValueValidator 检查数值最小值
	MinValueValidator struct {
		Min     any
		Message string
	}

	// MaxValueValidator 检查数值最大值
	MaxValueValidator struct {
		Max     any
		Message string
	}
)

// Run 实现CmdRunner接口
func (f CmdRunnerFunc) Run(cmd *cobra.Command, args []string) error {
	return f(cmd, args)
}
