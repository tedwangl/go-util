# Dagu 与 Go 集成指南

## 概述

Dagu 本身使用 YAML 配置工作流，但可以很好地与 Go 代码集成。

## 集成方式对比

| 方式 | 复杂度 | 灵活性 | 推荐度 |
|------|--------|--------|--------|
| **YAML + Go 程序** | ⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Go 生成 YAML** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **纯 Go (Temporal)** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

## 方式 1: YAML + Go 程序（最推荐）

### 优点
- ✅ 简单直接
- ✅ 充分利用 Dagu 的 Web UI
- ✅ Go 代码专注业务逻辑
- ✅ 易于调试和维护

### 示例

**Go 程序** (`myapp/main.go`):
```go
package main

import (
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("用法: myapp <command>")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "extract":
        extractData()
    case "transform":
        transformData()
    case "load":
        loadData()
    default:
        fmt.Printf("未知命令: %s\n", os.Args[1])
        os.Exit(1)
    }
}

func extractData() {
    fmt.Println("提取数据...")
    // 你的业务逻辑
}

func transformData() {
    fmt.Println("转换数据...")
    // 你的业务逻辑
}

func loadData() {
    fmt.Println("加载数据...")
    // 你的业务逻辑
}
```

**Dagu 工作流** (`~/.dagu/dags/myapp.yaml`):
```yaml
name: myapp-workflow
description: 使用 Go 程序的工作流

steps:
  - name: extract
    command: ./myapp extract
  
  - name: transform
    command: ./myapp transform
    depends:
      - extract
  
  - name: load
    command: ./myapp load
    depends:
      - transform
```

### 使用步骤

1. 编译 Go 程序：
```bash
go build -o myapp myapp/main.go
```

2. 创建 Dagu 工作流（如上）

3. 在 Web UI 中运行

---

## 方式 2: Go 生成 YAML

### 优点
- ✅ 动态生成工作流
- ✅ 可以根据条件生成不同的流程
- ✅ 类型安全
- ✅ 可复用

### 示例

见 `examples/go-generator/main.go`

**使用**:
```bash
go run examples/go-generator/main.go
```

这会在 `~/.dagu/dags/` 生成 YAML 文件。

### 适用场景
- 需要根据配置生成不同的工作流
- 有大量相似的工作流需要创建
- 需要程序化管理工作流

---

## 方式 3: Go 作为任务执行器

### 优点
- ✅ 统一的任务接口
- ✅ 易于测试
- ✅ 代码复用

### 示例

见 `examples/go-tasks/tasks.go`

**使用**:
```bash
# 单独测试任务
go run examples/go-tasks/tasks.go extract

# 通过 Dagu 运行
# 在 Web UI 中运行 go-tasks-workflow
```

---

## 实际应用示例

### 场景 1: 数据处理管道

**目录结构**:
```
myproject/
├── cmd/
│   └── pipeline/
│       └── main.go          # 主程序
├── internal/
│   ├── extract/
│   │   └── extract.go       # 提取逻辑
│   ├── transform/
│   │   └── transform.go     # 转换逻辑
│   └── load/
│       └── load.go          # 加载逻辑
└── dagu/
    └── pipeline.yaml        # Dagu 工作流
```

**main.go**:
```go
package main

import (
    "myproject/internal/extract"
    "myproject/internal/transform"
    "myproject/internal/load"
)

func main() {
    cmd := os.Args[1]
    
    switch cmd {
    case "extract":
        extract.Run()
    case "transform":
        transform.Run()
    case "load":
        load.Run()
    }
}
```

**pipeline.yaml**:
```yaml
name: data-pipeline
schedule: "0 1 * * *"

steps:
  - name: extract
    command: ./pipeline extract
  
  - name: transform
    command: ./pipeline transform
    depends: [extract]
  
  - name: load
    command: ./pipeline load
    depends: [transform]
```

### 场景 2: 微服务健康检查

**health-check.go**:
```go
package main

import (
    "fmt"
    "net/http"
    "os"
)

func checkService(url string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("status: %d", resp.StatusCode)
    }
    return nil
}

func main() {
    service := os.Args[1]
    urls := map[string]string{
        "api":      "http://localhost:8080/health",
        "database": "http://localhost:5432/health",
        "cache":    "http://localhost:6379/health",
    }
    
    if err := checkService(urls[service]); err != nil {
        fmt.Printf("❌ %s 健康检查失败: %v\n", service, err)
        os.Exit(1)
    }
    
    fmt.Printf("✅ %s 健康检查通过\n", service)
}
```

**health-check.yaml**:
```yaml
name: health-check
schedule: "*/5 * * * *"  # 每 5 分钟

steps:
  - name: check-api
    command: go run health-check.go api
  
  - name: check-database
    command: go run health-check.go database
  
  - name: check-cache
    command: go run health-check.go cache
  
  - name: report
    command: echo "所有服务健康"
    depends:
      - check-api
      - check-database
      - check-cache
```

---

## 与 Temporal 对比

### Dagu + Go

**优点**:
- ✅ 简单轻量
- ✅ 有 Web UI
- ✅ 快速上手
- ✅ 适合个人/小团队

**缺点**:
- ❌ 不支持纯 Go 定义工作流
- ❌ 功能相对简单
- ❌ 不适合复杂的分布式场景

**适合**:
- 数据处理管道
- 定时任务编排
- 简单的业务流程

### Temporal

**优点**:
- ✅ 纯 Go 代码定义
- ✅ 强大的状态管理
- ✅ 适合复杂业务流程
- ✅ 分布式支持

**缺点**:
- ❌ 学习曲线陡峭
- ❌ 部署复杂
- ❌ 对个人来说过重

**适合**:
- 微服务编排
- 长时间运行的流程
- 需要强一致性的场景

---

## 推荐方案

### 个人项目
**推荐**: Dagu + Go 程序

```yaml
# 简单直接
name: my-workflow
steps:
  - name: task1
    command: go run task1.go
  - name: task2
    command: go run task2.go
    depends: [task1]
```

### 小团队
**推荐**: Dagu + Go 生成器

```go
// 动态生成工作流
workflow := GenerateWorkflow(config)
workflow.SaveToFile("workflow.yaml")
```

### 企业应用
**推荐**: Temporal

```go
// 纯 Go 代码
func OrderWorkflow(ctx workflow.Context) error {
    workflow.ExecuteActivity(ctx, ProcessPayment)
    workflow.ExecuteActivity(ctx, ShipOrder)
    return nil
}
```

---

## 最佳实践

### 1. 保持简单

```yaml
# 好 ✅
steps:
  - name: process
    command: ./myapp process

# 避免 ❌
steps:
  - name: complex
    command: |
      cd /tmp
      git clone ...
      cd repo
      go build
      ./app
```

### 2. 使用编译后的二进制

```yaml
# 好 ✅ - 快速
steps:
  - name: task
    command: ./myapp task

# 避免 ❌ - 每次都编译
steps:
  - name: task
    command: go run main.go task
```

### 3. 错误处理

```go
// Go 程序中
func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "错误: %v\n", err)
        os.Exit(1)  // 重要：返回非零退出码
    }
}
```

### 4. 日志输出

```go
// 输出到 stdout，Dagu 会捕获
fmt.Println("处理中...")
fmt.Printf("进度: %d%%\n", progress)
```

---

## 总结

**Dagu 不支持纯 Go 代码定义工作流**，但可以很好地与 Go 程序集成：

1. **最简单**: YAML 调用 Go 程序
2. **更灵活**: Go 生成 YAML
3. **需要纯 Go**: 使用 Temporal

对于大多数个人和小团队场景，**Dagu + Go 程序**是最佳选择！
