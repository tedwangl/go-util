package cobrax

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SetConfig 设置配置文件并初始化viper
func (t *Tool) SetConfig(cfgFile string) {
	originalPreRunE := t.rootCmd.PersistentPreRunE

	t.rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// 从命令行标志获取配置文件路径
		flagConfig, _ := cmd.Flags().GetString("config")
		if flagConfig != "" {
			cfgFile = flagConfig
		}

		// 1. 读取配置文件（可选）
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
			_ = viper.ReadInConfig() // 忽略配置文件不存在的错误
		}

		// 2. 启用环境变量
		viper.SetEnvPrefix(t.envPrefix)
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		viper.AutomaticEnv()

		// 3. 绑定所有标志到 viper
		if err := bindAllFlags(cmd); err != nil {
			return err
		}

		// 4. 执行原有的 PreRunE（如果存在）
		if originalPreRunE != nil {
			return originalPreRunE(cmd, args)
		}

		return nil
	}
}

// bindAllFlags 递归绑定命令及其父命令的所有标志
func bindAllFlags(cmd *cobra.Command) error {
	// 绑定当前命令的标志
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("绑定命令标志失败: %w", err)
	}

	// 绑定继承的标志（包括父命令的 PersistentFlags）
	if err := viper.BindPFlags(cmd.InheritedFlags()); err != nil {
		return fmt.Errorf("绑定继承标志失败: %w", err)
	}

	return nil
}

// IsConfigSet 检查配置是否设置
func (t *Tool) IsConfigSet(key string) bool {
	return viper.IsSet(key)
}

// UnmarshalConfig 将配置绑定到结构体
func (t *Tool) UnmarshalConfig(target any) error {
	return viper.Unmarshal(target)
}
