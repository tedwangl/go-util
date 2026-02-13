# Dagu 快速上手 🚀

## ✅ 已完成

- ✅ Dagu 已安装
- ✅ 服务已启动
- ✅ 示例工作流已创建

## 🌐 访问 Web UI

打开浏览器访问：**http://localhost:8080**

你会看到：
- 📊 工作流列表
- ▶️ 运行按钮
- 📈 执行历史
- 📝 日志查看

## 📁 工作流文件位置

```bash
~/.dagu/dags/
├── hello.yaml              # Hello World 示例
├── data-pipeline.yaml      # 数据管道示例
├── parallel-tasks.yaml     # 并行任务示例
└── retry-example.yaml      # 重试机制示例
```

## 🎯 快速操作

### 1. 查看所有工作流

在 Web UI 中，你会看到 4 个示例工作流。

### 2. 手动运行工作流

1. 点击工作流名称
2. 点击右上角的 "Run" 按钮
3. 查看实时执行状态和日志

### 3. 查看 DAG 图

点击工作流 → "DAG" 标签页，可以看到任务依赖关系的可视化图。

### 4. 查看执行历史

点击工作流 → "History" 标签页，查看所有历史执行记录。

## 📝 创建你的第一个工作流

### 方法 1: 通过文件

```bash
cat > ~/.dagu/dags/my-first-workflow.yaml << 'EOF'
name: my-first-workflow
description: 我的第一个工作流

steps:
  - name: step1
    command: echo "第一步完成"
  
  - name: step2
    command: echo "第二步完成"
    depends:
      - step1
EOF
```

刷新 Web UI，新工作流会自动出现。

### 方法 2: 通过 Web UI

1. 点击右上角 "New DAG"
2. 填写工作流名称
3. 编辑 YAML 配置
4. 保存

## 🔧 常用命令

### 启动 Dagu

```bash
dagu start-all
```

### 停止 Dagu

```bash
# 按 Ctrl+C 或
pkill dagu
```

### 查看帮助

```bash
dagu --help
```

### 验证工作流配置

```bash
dagu validate ~/.dagu/dags/hello.yaml
```

## 📚 工作流示例

### 简单任务

```yaml
name: simple-task
steps:
  - name: hello
    command: echo "Hello, World!"
```

### 带依赖的任务

```yaml
name: with-dependencies
steps:
  - name: task1
    command: echo "Task 1"
  
  - name: task2
    command: echo "Task 2"
    depends:
      - task1
  
  - name: task3
    command: echo "Task 3"
    depends:
      - task2
```

### 并行任务

```yaml
name: parallel
steps:
  - name: task-a
    command: echo "Task A"
  
  - name: task-b
    command: echo "Task B"
  
  - name: task-c
    command: echo "Task C"
  
  - name: final
    command: echo "All done"
    depends:
      - task-a
      - task-b
      - task-c
```

### 定时执行

```yaml
name: scheduled
schedule: "0 1 * * *"  # 每天凌晨 1 点
steps:
  - name: backup
    command: ./backup.sh
```

### 带重试

```yaml
name: with-retry
steps:
  - name: unstable-task
    command: ./unstable-script.sh
    retryPolicy:
      limit: 3
      intervalSec: 10
```

### 使用环境变量

```yaml
name: with-env
env:
  - API_KEY: your-api-key
  - DATABASE_URL: postgresql://localhost/mydb

steps:
  - name: task
    command: python script.py
```

## 🎨 Web UI 功能

### 主页面
- 📋 工作流列表
- 🔍 搜索过滤
- ▶️ 快速运行
- 📊 状态概览

### 工作流详情
- 📈 DAG 可视化
- 📝 实时日志
- 📅 执行历史
- ⚙️ 配置查看
- ✏️ 在线编辑

### 执行历史
- ✅ 成功/失败状态
- ⏱️ 执行时间
- 📋 详细日志
- 🔄 重新运行

## 💡 实用技巧

### 1. 调试工作流

在 Web UI 中点击 "Run" 后，可以实时查看每个步骤的输出。

### 2. 快速测试

创建一个简单的测试工作流：

```yaml
name: test
steps:
  - name: test
    command: echo "测试成功"
```

### 3. 使用脚本

```yaml
name: run-script
steps:
  - name: python-script
    command: python /path/to/script.py
  
  - name: shell-script
    command: bash /path/to/script.sh
```

### 4. 条件执行

```yaml
name: conditional
steps:
  - name: check
    command: test -f /tmp/flag.txt
  
  - name: if-exists
    command: echo "文件存在"
    depends:
      - check
    preconditions:
      - condition: $CHECK_EXIT_CODE
        expected: "0"
```

## 🚨 常见问题

### Q: 工作流没有出现在列表中？

A: 检查 YAML 语法是否正确，查看 Dagu 日志中的错误信息。

### Q: 如何停止正在运行的工作流？

A: 在 Web UI 中点击工作流，然后点击 "Stop" 按钮。

### Q: 如何修改工作流？

A: 直接编辑 `~/.dagu/dags/` 下的 YAML 文件，Dagu 会自动重新加载。

### Q: 日志在哪里？

A: 在 Web UI 的工作流详情页面可以查看实时日志。

## 🎉 下一步

1. ✅ 浏览 Web UI：http://localhost:8080
2. ✅ 运行示例工作流
3. ✅ 创建你自己的工作流
4. ✅ 查看执行日志和历史

## 📖 更多资源

- [官方文档](https://dagu.readthedocs.io/)
- [GitHub](https://github.com/dagu-org/dagu)
- [示例集合](https://github.com/dagu-org/dagu/tree/main/examples)

---

**享受使用 Dagu！** 🎊

如果有任何问题，查看日志或访问官方文档。
