# Workflow 工作流引擎集成

本目录包含两种工作流引擎的 Docker 集成方案：

## 1. Airflow - 数据管道编排

适用场景：
- ETL/ELT 数据管道
- 批处理任务调度
- 数据工程工作流
- 定时任务编排

特点：
- Python 原生，DAG 定义
- 丰富的 Operators
- 强大的调度能力
- Web UI 可视化

## 2. Temporal - 微服务编排

适用场景：
- 长时间运行的业务流程
- 微服务编排
- 分布式事务
- 订单/支付流程

特点：
- 多语言支持（Go/Python/Java/TypeScript）
- 自动重试和状态管理
- 事件溯源
- 强一致性保证

## 快速开始

### Airflow
```bash
cd workflow/airflow
docker-compose up -d
# 访问 http://localhost:8080
# 用户名: airflow
# 密码: airflow
```

### Temporal
```bash
cd workflow/temporal
docker-compose up -d
# 访问 http://localhost:8088 (Web UI)
```

## 目录结构

```
workflow/
├── README.md
├── airflow/
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── dags/              # DAG 定义文件
│   ├── plugins/           # 自定义插件
│   ├── config/            # 配置文件
│   └── logs/              # 日志目录
└── temporal/
    ├── docker-compose.yml
    ├── workflows/         # Go 工作流定义
    ├── activities/        # Go 活动定义
    └── config/            # 配置文件
```

## 对比

| 特性 | Airflow | Temporal |
|------|---------|----------|
| 语言 | Python | Go/Python/Java/TS |
| 适用场景 | 数据管道 | 业务流程 |
| 调度 | Cron 表达式 | 事件驱动 |
| 状态管理 | 数据库 | 事件溯源 |
| 重试机制 | 配置化 | 自动化 |
| 学习曲线 | 中等 | 较陡 |
| 部署复杂度 | 中等 | 较高 |

## 选择建议

- **数据工程团队**：选择 Airflow
- **微服务团队**：选择 Temporal
- **混合场景**：两者都部署，各司其职
