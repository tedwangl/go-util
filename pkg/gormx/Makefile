.PHONY: help docker-up docker-down docker-logs test-all test-single test-master-slave test-mixed test-pure clean

help: ## 显示帮助信息
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

docker-up: ## 启动所有 Docker 容器
	@echo "🚀 启动 Docker 容器..."
	docker-compose up -d
	@echo "⏳ 等待数据库就绪..."
	@sleep 15
	@echo "✅ 所有容器已启动"
	@docker-compose ps

docker-down: ## 停止并删除所有容器
	@echo "🛑 停止 Docker 容器..."
	docker-compose down -v
	@echo "✅ 所有容器已停止"

docker-logs: ## 查看容器日志
	docker-compose logs -f

docker-ps: ## 查看容器状态
	docker-compose ps

test-all: ## 运行所有集成测试
	@echo "🧪 运行所有集成测试..."
	go test -v -run TestIntegration

test-single: ## 测试场景 1：单库
	@echo "🧪 测试场景 1：单库..."
	go test -v -run TestIntegration_Scenario1_SingleDB

test-master-slave: ## 测试场景 2：主从
	@echo "🧪 测试场景 2：主从..."
	go test -v -run TestIntegration_Scenario2_MasterSlave

test-mixed: ## 测试场景 3：混合分片
	@echo "🧪 测试场景 3：混合分片..."
	go test -v -run TestIntegration_Scenario3_MixedSharding

test-pure: ## 测试场景 4：纯分片
	@echo "🧪 测试场景 4：纯分片..."
	go test -v -run TestIntegration_Scenario4_PureSharding

clean: ## 清理测试数据和容器
	@echo "🧹 清理环境..."
	docker-compose down -v
	@echo "✅ 清理完成"

setup: docker-up ## 完整设置（启动容器 + 运行测试）
	@echo "⏳ 等待数据库完全就绪..."
	@sleep 5
	@echo "🧪 运行测试..."
	@make test-all

quick-test: ## 快速测试（假设容器已启动）
	@echo "🧪 快速测试..."
	go test -v -run TestIntegration -timeout 30s
