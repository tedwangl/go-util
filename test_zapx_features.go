package main

import (
	"fmt"

	zap "github.com/tedwangl/go-util/pkg/logger/zapx"
)

// 实现 Sensitive 接口的示例类型
type User struct {
	Name     string
	Password string
	Email    string
}

func (u User) MaskSensitive() any {
	// 返回 map[string]any，这样 zap 库在序列化时就会使用这个 map
	return map[string]any{
		"name":     u.Name,
		"password": "****", // 脱敏处理
		"email":    u.Email,
	}
}

func main() {
	// 配置日志
	config := zap.LogConf{
		Mode:                "console",
		Encoding:            "console",
		Level:               "debug",
		StackCooldownMillis: 1000,
	}

	// 初始化日志
	err := zap.SetUp(config)
	if err != nil {
		fmt.Printf("SetUp failed: %v\n", err)
		return
	}
	defer zap.Close()

	// 测试基本日志
	fmt.Println("=== Testing basic logs ===")
	zap.Info("This is an info message")
	zap.Error("This is an error message")
	zap.Debug("This is a debug message")
	zap.Slow("This is a slow message")
	zap.Severe("This is a severe message")
	zap.Stat("This is a stat message")
	zap.Alert("This is an alert message")

	// 测试错误栈
	fmt.Println("\n=== Testing error stack ===")
	zap.ErrorStack("This is an error stack message")

	// 测试敏感信息
	fmt.Println("\n=== Testing sensitive information ===")
	user := User{
		Name:     "John Doe",
		Password: "secret123",
		Email:    "john.doe@example.com",
	}

	// 直接传递 User 对象，测试敏感信息处理
	zap.Info("User info:", zap.LogField{Key: "user", Value: user})

	// 测试敏感信息作为日志消息
	zap.Info("Sensitive user:", user)

	// 测试调用者信息
	fmt.Println("\n=== Testing caller information ===")
	// 调用一个函数，然后在函数中记录日志
	logFromFunction()

	// 测试时间格式
	testTimeFormat()

	fmt.Println("\nTest done!")
}

func logFromFunction() {
	// 这里的日志应该包含调用者信息，显示来自 logFromFunction
	zap.Info("Log from function")
}

func testTimeFormat() {
	// 测试默认时间格式
	fmt.Println("\n=== Testing default time format ===")
	config1 := zap.LogConf{
		Mode:     "console",
		Encoding: "console",
		Level:    "debug",
	}

	// 重置日志设置
	zap.Reset()

	err := zap.SetUp(config1)
	if err != nil {
		fmt.Printf("SetUp failed: %v\n", err)
		return
	}

	zap.Info("This is a test with default time format")

	// 测试自定义时间格式
	fmt.Println("\n=== Testing custom time format ===")
	config2 := zap.LogConf{
		Mode:       "console",
		Encoding:   "console",
		Level:      "debug",
		TimeFormat: "2006-01-02 15:04:05", // 自定义时间格式
	}

	// 重置日志设置
	zap.Reset()

	err = zap.SetUp(config2)
	if err != nil {
		fmt.Printf("SetUp failed: %v\n", err)
		return
	}

	zap.Info("This is a test with custom time format")
}
