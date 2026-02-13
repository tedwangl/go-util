package cobrax

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// DefaultErrorHandler 默认错误处理函数
func DefaultErrorHandler(err error, cmd *cobra.Command) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n使用方法:\n")
		cmd.Help()
	}
	return err
}

// LoggingErrorHandler 带日志记录的错误处理函数
func LoggingErrorHandler(logger *zap.Logger) ErrorHandler {
	return func(err error, cmd *cobra.Command) error {
		if err != nil {
			logger.Error("命令执行失败",
				zap.String("command", cmd.CommandPath()),
				zap.Error(err),
			)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "\n使用方法:\n")
			cmd.Help()
		}
		return err
	}
}
