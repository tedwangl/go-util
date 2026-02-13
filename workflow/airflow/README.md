# Airflow 工作流引擎

Apache Airflow 是一个用于编排复杂计算工作流和数据处理管道的平台。

## 快速开始

### 1. 启动服务

```bash
# 创建必要的目录
mkdir -p ./dags ./logs ./plugins ./config

# 启动 Airflow
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 2. 访问 Web UI

- URL: http://localhost:8080
- 用户名: `airflow`
- 密码: `airflow`

### 3. 停止服务

```bash
docker-compose down

# 清理所有数据（包括数据库）
docker-compose down -v
```

## 目录结构

```
airflow/
├── docker-compose.yml    # Docker 编排配置
├── .env                  # 环境变量
├── dags/                 # DAG 定义文件（Python）
│   ├── example_dag.py
│   └── parallel_tasks_dag.py
├── logs/                 # 日志目录（自动生成）
├── plugins/              # 自定义插件
└── config/               # 配置文件
```

## DAG 示例

### 简单 ETL 流程

```python
from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime

def extract():
    print("提取数据")

def transform():
    print("转换数据")

def load():
    print("加载数据")

with DAG('simple_etl', start_date=datetime(2024, 1, 1)) as dag:
    extract_task = PythonOperator(task_id='extract', python_callable=extract)
    transform_task = PythonOperator(task_id='transform', python_callable=transform)
    load_task = PythonOperator(task_id='load', python_callable=load)
    
    extract_task >> transform_task >> load_task
```

## 常用操作

### 触发 DAG

```bash
# 通过 CLI
docker-compose exec airflow-webserver airflow dags trigger example_etl_pipeline

# 或在 Web UI 中点击 "Trigger DAG" 按钮
```

### 查看 DAG 列表

```bash
docker-compose exec airflow-webserver airflow dags list
```

### 测试单个任务

```bash
docker-compose exec airflow-webserver airflow tasks test example_etl_pipeline extract 2024-01-01
```

## 配置说明

### 调度间隔

- `@once`: 只运行一次
- `@hourly`: 每小时
- `@daily`: 每天
- `@weekly`: 每周
- `@monthly`: 每月
- `timedelta(hours=1)`: 自定义间隔
- `'0 0 * * *'`: Cron 表达式

### 重试策略

```python
default_args = {
    'retries': 3,                          # 重试次数
    'retry_delay': timedelta(minutes=5),   # 重试间隔
    'retry_exponential_backoff': True,     # 指数退避
}
```

## 常见问题

### 1. 权限问题

```bash
# 设置正确的 UID
echo -e "AIRFLOW_UID=$(id -u)" > .env
```

### 2. DAG 不显示

- 检查 DAG 文件语法错误
- 查看 scheduler 日志：`docker-compose logs airflow-scheduler`
- 刷新 DAG：等待约 30 秒

### 3. 任务失败

- 查看任务日志：Web UI -> DAG -> Task -> Log
- 检查依赖是否安装
- 验证数据库连接

## 最佳实践

1. **使用 XCom 传递数据**：任务间传递小量数据
2. **避免在 DAG 文件中执行重操作**：DAG 文件会被频繁解析
3. **使用连接和变量**：敏感信息存储在 Airflow 的 Connections/Variables
4. **合理设置并发**：避免资源耗尽
5. **监控和告警**：配置邮件或 Slack 通知

## 扩展阅读

- [官方文档](https://airflow.apache.org/docs/)
- [最佳实践](https://airflow.apache.org/docs/apache-airflow/stable/best-practices.html)
- [Operators 列表](https://airflow.apache.org/docs/apache-airflow/stable/operators-and-hooks-ref.html)
