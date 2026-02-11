package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	redisxerrors "github.com/tedwangl/go-util/pkg/redisx/errors"
	"gopkg.in/yaml.v3"
)

// ErrConfigMultiMasterAddr 多主多从地址为空
func ErrConfigMultiMasterAddr(index int) error {
	return redisxerrors.NewConfigError(fmt.Sprintf("masters[%d]", index), "address is empty", nil)
}

// Loader 配置加载器接口
type Loader interface {
	Load() (*Config, error)
}

// FileLoader 文件加载器
type FileLoader struct {
	Path string
}

// NewFileLoader 创建文件加载器
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{Path: path}
}

// Load 从文件加载配置
func (l *FileLoader) Load() (*Config, error) {
	if l.Path == "" {
		return nil, errors.New("file path is empty")
	}

	content, err := os.ReadFile(l.Path)
	if err != nil {
		return nil, fmt.Errorf("read file error: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(l.Path))
	cfg := &Config{}

	switch ext {
	case ".json":
		err = json.Unmarshal(content, cfg)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(content, cfg)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// EnvLoader 环境变量加载器
type EnvLoader struct {
	Prefix string
}

// NewEnvLoader 创建环境变量加载器
func NewEnvLoader(prefix string) *EnvLoader {
	return &EnvLoader{Prefix: prefix}
}

// Load 从环境变量加载配置
func (l *EnvLoader) Load() (*Config, error) {
	cfg := DefaultConfig()

	prefix := l.Prefix
	if prefix == "" {
		prefix = "REDISX"
	}

	if mode := os.Getenv(fmt.Sprintf("%s_MODE", prefix)); mode != "" {
		cfg.Mode = mode
	}

	if debug := os.Getenv(fmt.Sprintf("%s_DEBUG", prefix)); debug == "true" {
		cfg.Debug = true
	}

	if password := os.Getenv(fmt.Sprintf("%s_PASSWORD", prefix)); password != "" {
		cfg.Password = password
	}

	switch cfg.Mode {
	case "single":
		if cfg.Single == nil {
			cfg.Single = &SingleConfig{}
		}
		if addr := os.Getenv(fmt.Sprintf("%s_SINGLE_ADDR", prefix)); addr != "" {
			cfg.Single.Addr = addr
		}
	case "sentinel":
		if cfg.Sentinel == nil {
			cfg.Sentinel = &SentinelConfig{}
		}
		if masterName := os.Getenv(fmt.Sprintf("%s_SENTINEL_MASTER_NAME", prefix)); masterName != "" {
			cfg.Sentinel.MasterName = masterName
		}
		if addrs := os.Getenv(fmt.Sprintf("%s_SENTINEL_ADDRS", prefix)); addrs != "" {
			cfg.Sentinel.SentinelAddrs = strings.Split(addrs, ",")
		}
	case "cluster":
		if cfg.Cluster == nil {
			cfg.Cluster = &ClusterConfig{}
		}
		if addrs := os.Getenv(fmt.Sprintf("%s_CLUSTER_ADDRS", prefix)); addrs != "" {
			cfg.Cluster.Addrs = strings.Split(addrs, ",")
		}
	case "multi-master":
		if cfg.MultiMaster == nil {
			cfg.MultiMaster = &MultiMasterConfig{}
		}
		if addrs := os.Getenv(fmt.Sprintf("%s_MULTI_MASTER_ADDRS", prefix)); addrs != "" {
			addrList := strings.Split(addrs, ",")
			for _, addr := range addrList {
				if addr != "" {
					cfg.MultiMaster.Masters = append(cfg.MultiMaster.Masters, MasterConfig{Addr: addr})
				}
			}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	loader := NewFileLoader(path)
	return loader.Load()
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv(prefix string) (*Config, error) {
	loader := NewEnvLoader(prefix)
	return loader.Load()
}

// LoadFromBytes 从字节数组加载配置
func LoadFromBytes(data []byte, format string) (*Config, error) {
	cfg := &Config{}

	switch strings.ToLower(format) {
	case "json":
		err := json.Unmarshal(data, cfg)
		if err != nil {
			return nil, err
		}
	case "yaml", "yml":
		err := yaml.Unmarshal(data, cfg)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
