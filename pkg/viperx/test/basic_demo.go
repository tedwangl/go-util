package main

import (
	"fmt"
	"log"

	"github.com/tedwangl/go-util/pkg/viperx"
)

type AppConfig struct {
	Server struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`
}

func main() {
	fmt.Println("=== Viper 配置管理演示 ===\n")

	// 方式 1: 最简单 - 直接加载文件
	var config1 AppConfig
	if err := viperx.LoadFromFile("config.yaml", &config1); err != nil {
		log.Printf("方式1失败（正常，文件不存在）: %v\n", err)
	}

	// 方式 2: 文件 + 环境变量
	var config2 AppConfig
	if err := viperx.LoadWithEnv("config.yaml", "APP", &config2); err != nil {
		log.Printf("方式2失败（正常）: %v\n", err)
	}

	// 方式 3: 完整配置
	c, err := viperx.New(
		viperx.WithName("config"),
		viperx.WithPath(".", "./config"),
		viperx.WithEnvPrefix("APP"),
		viperx.WithDefaults(map[string]any{
			"server.host": "localhost",
			"server.port": 8080,
		}),
		viperx.WithOnChange(func() {
			fmt.Println("配置文件变化了！")
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Load(); err != nil {
		log.Printf("加载配置: %v\n", err)
	}

	// 读取配置
	fmt.Printf("Server Host: %s\n", c.GetString("server.host"))
	fmt.Printf("Server Port: %d\n", c.GetInt("server.port"))

	// 解析到结构体
	var config AppConfig
	if err := c.Unmarshal(&config); err != nil {
		log.Printf("解析配置: %v\n", err)
	} else {
		fmt.Printf("完整配置: %+v\n", config)
	}

	fmt.Println("\n演示完成")
}
