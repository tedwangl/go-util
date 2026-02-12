package viperx

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type (
	// Config 配置管理器
	Config struct {
		v        *viper.Viper
		onChange func()
	}

	// Option 配置选项
	Option func(*Config)
)

// New 创建配置管理器
// 自动处理：配置文件 + 环境变量 + 文件监听
//
// 默认行为：
// - 配置文件：./config.yaml
// - 环境变量：自动绑定（前缀可选）
// - 文件监听：自动启用
func New(opts ...Option) (*Config, error) {
	c := &Config{
		v: viper.New(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// WithFile 指定配置文件
// 支持格式：yaml, json, toml, ini, env
func WithFile(path string) Option {
	return func(c *Config) {
		c.v.SetConfigFile(path)
	}
}

// WithName 指定配置文件名（不含扩展名）
// 例如：WithName("config") 会查找 config.yaml, config.json 等
func WithName(name string) Option {
	return func(c *Config) {
		c.v.SetConfigName(name)
	}
}

// WithPath 指定配置文件搜索路径
func WithPath(paths ...string) Option {
	return func(c *Config) {
		for _, path := range paths {
			c.v.AddConfigPath(path)
		}
	}
}

// WithEnvPrefix 设置环境变量前缀
// 例如：WithEnvPrefix("APP") 会读取 APP_DATABASE_HOST 等
func WithEnvPrefix(prefix string) Option {
	return func(c *Config) {
		c.v.SetEnvPrefix(prefix)
		c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		c.v.AutomaticEnv()
	}
}

// WithDefaults 设置默认值
func WithDefaults(defaults map[string]any) Option {
	return func(c *Config) {
		for key, value := range defaults {
			c.v.SetDefault(key, value)
		}
	}
}

// WithOnChange 设置配置变化回调
func WithOnChange(callback func()) Option {
	return func(c *Config) {
		c.onChange = callback
	}
}

// Load 加载配置
func (c *Config) Load() error {
	// 读取配置文件
	if err := c.v.ReadInConfig(); err != nil {
		// 配置文件不存在不算错误
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	// 启动文件监听
	if c.onChange != nil {
		c.v.WatchConfig()
		c.v.OnConfigChange(func(e fsnotify.Event) {
			c.onChange()
		})
	}

	return nil
}

// Unmarshal 解析配置到结构体
func (c *Config) Unmarshal(v any) error {
	return c.v.Unmarshal(v)
}

// UnmarshalKey 解析指定 key 的配置
func (c *Config) UnmarshalKey(key string, v any) error {
	return c.v.UnmarshalKey(key, v)
}

// Get 获取配置值
func (c *Config) Get(key string) any {
	return c.v.Get(key)
}

// GetString 获取字符串配置
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt 获取整数配置
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool 获取布尔配置
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetStringSlice 获取字符串数组配置
func (c *Config) GetStringSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// Set 设置配置值（运行时）
func (c *Config) Set(key string, value any) {
	c.v.Set(key, value)
}

// Viper 获取底层 viper 实例（高级用户）
func (c *Config) Viper() *viper.Viper {
	return c.v
}

// LoadFromFile 快捷方法：从文件加载配置
func LoadFromFile(path string, v any) error {
	c, err := New(WithFile(path))
	if err != nil {
		return err
	}

	if err := c.Load(); err != nil {
		return err
	}

	return c.Unmarshal(v)
}

// LoadWithEnv 快捷方法：从文件 + 环境变量加载配置
func LoadWithEnv(path, envPrefix string, v any) error {
	c, err := New(
		WithFile(path),
		WithEnvPrefix(envPrefix),
	)
	if err != nil {
		return err
	}

	if err := c.Load(); err != nil {
		return err
	}

	return c.Unmarshal(v)
}
