.PHONY: help start stop restart logs worker starter clean

help:
	@echo "Temporal 工作流管理命令"
	@echo ""
	@echo "使用方法:"
	@echo "  make start      - 启动 Temporal 服务"
	@echo "  make stop       - 停止 Temporal 服务"
	@echo "  make restart    - 重启 Temporal 服务"
	@echo "  make logs       - 查看服务日志"
	@echo "  make worker     - 启动 Worker"
	@echo "  make starter    - 运行工作流启动器"
	@echo "  make clean      - 清理所有数据"

start:
	@echo "启动 Temporal 服务..."
	docker-compose up -d
	@echo "等待服务就绪..."
	@sleep 10
	@echo "服务已启动！访问 http://localhost:8088 查看 Web UI"

stop:
	@echo "停止 Temporal 服务..."
	docker-compose down

restart:
	@echo "重启 Temporal 服务..."
	docker-compose restart

logs:
	docker-compose logs -f

worker:
	@echo "启动 Worker..."
	go run worker/main.go

starter:
	@echo "运行工作流启动器..."
	go run starter/main.go

clean:
	@echo "清理所有数据..."
	docker-compose down -v
	@echo "数据已清理！"
