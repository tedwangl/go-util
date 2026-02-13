# Dagu - 轻量级工作流引擎

最适合个人使用的工作流引擎，带 Web UI！

## 特点

- ✅ 单二进制文件，一键启动
- ✅ 美观的 Web UI
- ✅ YAML 定义工作流
- ✅ 支持 DAG 依赖
- ✅ 定时调度
- ✅ 执行历史和日志
- ✅ 轻量级，资源占用少

## 安装

### macOS
```bash
brew install dagu-org/brew/dagu
```

### 或使用 Go 安装
```bash
go install github.com/dagu-org/dagu@latest
```

### 或下载二进制
访问 https://github.com/dagu-org/dagu/releases

## 快速开始

### 1. 启动 Dagu

```bash
dagu start-all
```

访问：http://localhost:8080

### 2. 创建工作流

创建文件 `~/.dagu/dags/example.yaml`:

```yaml
name: example
schedule: "0 1 * * *"  # 每天凌晨 1 点

steps:
  - name: step1
    command: echo "Hello from step1"
  
  - name: step2
    command: echo "Hello from step2"
    depends:
      - step1
  
  - name: step3
    command: echo "Hello from step3"
    depends:
      - step2
```

### 3. 在 Web UI 中查看和运行

打开 http://localhost:8080，你会看到：
- 工作流列表
- DAG 可视化
- 执行历史
- 实时日志

## 示例工作流

### 数据处理管道

```yaml
name: data-pipeline
description: 每日数据处理流程
schedule: "0 1 * * *"

env:
  - DATA_DIR: /data
  - OUTPUT_DIR: /output

steps:
  - name: extract
    command: python scripts/extract.py
    output: EXTRACT_RESULT
  
  - name: validate
    command: python scripts/validate.py
    depends:
      - extract
    preconditions:
      - condition: "`echo $EXTRACT_RESULT`"
        expected: "success"
  
  - name: transform
    command: python scripts/transform.py
    depends:
      - validate
  
  - name: load
    command: python scripts/load.py
    depends:
      - transform
  
  - name: notify
    command: |
      curl -X POST https://api.example.com/notify \
        -d '{"status": "completed"}'
    depends:
      - load
```

### 并行任务

```yaml
name: parallel-tasks
description: 并行处理多个任务

steps:
  - name: task1
    command: python task1.py
  
  - name: task2
    command: python task2.py
  
  - name: task3
    command: python task3.py
  
  - name: aggregate
    command: python aggregate.py
    depends:
      - task1
      - task2
      - task3
```

### 带重试的任务

```yaml
name: retry-example
description: 失败自动重试

steps:
  - name: api-call
    command: curl https://api.example.com/data
    retryPolicy:
      limit: 3
      intervalSec: 10
  
  - name: process
    command: python process.py
    depends:
      - api-call
```

### 条件执行

```yaml
name: conditional
description: 根据条件执行不同分支

steps:
  - name: check
    command: python check_condition.py
    output: CONDITION
  
  - name: branch-a
    command: echo "执行分支 A"
    depends:
      - check
    preconditions:
      - condition: "`echo $CONDITION`"
        expected: "A"
  
  - name: branch-b
    command: echo "执行分支 B"
    depends:
      - check
    preconditions:
      - condition: "`echo $CONDITION`"
        expected: "B"
```

## 高级功能

### 1. 邮件通知

```yaml
name: with-notification
description: 完成后发送邮件

mailOn:
  failure: true
  success: true

steps:
  - name: task
    command: python task.py
```

### 2. 超时控制

```yaml
steps:
  - name: long-task
    command: python long_task.py
    timeout: 3600  # 1 小时超时
```

### 3. 环境变量

```yaml
env:
  - API_KEY: ${API_KEY}
  - DATABASE_URL: postgresql://localhost/mydb

steps:
  - name: task
    command: python task.py
```

### 4. 子工作流

```yaml
steps:
  - name: sub-workflow
    run: another-dag
    params: "param1=value1 param2=value2"
```

## Web UI 功能

### 主界面
- 📊 工作流列表
- 🔍 搜索和过滤
- ▶️ 手动触发
- 📅 调度管理

### 工作流详情
- 📈 DAG 可视化图
- 📝 执行历史
- 📋 实时日志
- ⚙️ 配置查看

### 监控
- ✅ 成功/失败统计
- ⏱️ 执行时间趋势
- 🔔 告警配置

## 与其他方案对比

| 特性 | Dagu | Airflow | Temporal |
|------|------|---------|----------|
| 安装 | 单文件 | Docker | Docker |
| Web UI | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| 学习曲线 | 低 | 中 | 高 |
| 资源占用 | 低 | 中 | 高 |
| 适合场景 | 个人/小团队 | 数据工程 | 微服务 |

## 实际使用建议

### 个人项目
```bash
# 1. 安装 Dagu
brew install dagu-org/brew/dagu

# 2. 创建工作流目录
mkdir -p ~/.dagu/dags

# 3. 创建你的第一个工作流
cat > ~/.dagu/dags/hello.yaml << 'EOF'
name: hello
schedule: "*/5 * * * *"
steps:
  - name: greet
    command: echo "Hello, Dagu!"
EOF

# 4. 启动
dagu start-all

# 5. 访问 http://localhost:8080
```

### 集成到 devtool

可以将 Dagu 集成到你的 devtool 中：

```bash
# devtool 命令包装
devtool workflow start    # 启动 Dagu
devtool workflow stop     # 停止 Dagu
devtool workflow ui       # 打开 Web UI
devtool workflow create   # 创建新工作流
```

## 总结

**Dagu 是个人使用的最佳选择：**

✅ 优点：
- 轻量级，单文件
- 有 Web UI
- YAML 配置简单
- 功能够用
- 资源占用少

❌ 缺点：
- 功能不如 Airflow 丰富
- 社区相对较小
- 不适合大规模分布式场景

**推荐指数：⭐⭐⭐⭐⭐**

对于个人使用，Dagu 是完美的选择！
