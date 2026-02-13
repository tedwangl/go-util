#!/bin/bash

# Workflow 引擎快速启动脚本

set -e

echo "=========================================="
echo "  Workflow 引擎快速启动"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 显示菜单
show_menu() {
    echo "请选择要启动的工作流引擎："
    echo ""
    echo "1) Airflow  - 数据管道编排"
    echo "2) Temporal - 微服务编排"
    echo "3) 两者都启动"
    echo "4) 停止所有服务"
    echo "5) 退出"
    echo ""
}

# 启动 Airflow
start_airflow() {
    echo -e "${GREEN}启动 Airflow...${NC}"
    cd airflow
    
    # 创建必要的目录
    mkdir -p ./dags ./logs ./plugins ./config
    
    # 设置权限
    echo "AIRFLOW_UID=$(id -u)" > .env
    echo "_AIRFLOW_WWW_USER_USERNAME=airflow" >> .env
    echo "_AIRFLOW_WWW_USER_PASSWORD=airflow" >> .env
    
    # 启动服务
    docker-compose up -d
    
    echo -e "${GREEN}Airflow 启动成功！${NC}"
    echo "Web UI: http://localhost:8080"
    echo "用户名: airflow"
    echo "密码: airflow"
    echo ""
    
    cd ..
}

# 启动 Temporal
start_temporal() {
    echo -e "${GREEN}启动 Temporal...${NC}"
    cd temporal
    
    # 启动服务
    docker-compose up -d
    
    echo -e "${GREEN}Temporal 启动成功！${NC}"
    echo "Web UI: http://localhost:8088"
    echo ""
    echo "运行示例："
    echo "  cd temporal"
    echo "  go run worker/main.go    # 终端 1"
    echo "  go run starter/main.go   # 终端 2"
    echo ""
    
    cd ..
}

# 停止所有服务
stop_all() {
    echo -e "${YELLOW}停止所有服务...${NC}"
    
    if [ -d "airflow" ]; then
        cd airflow
        docker-compose down
        cd ..
        echo "Airflow 已停止"
    fi
    
    if [ -d "temporal" ]; then
        cd temporal
        docker-compose down
        cd ..
        echo "Temporal 已停止"
    fi
    
    echo -e "${GREEN}所有服务已停止${NC}"
}

# 主循环
while true; do
    show_menu
    read -p "请输入选项 [1-5]: " choice
    
    case $choice in
        1)
            start_airflow
            ;;
        2)
            start_temporal
            ;;
        3)
            start_airflow
            start_temporal
            ;;
        4)
            stop_all
            ;;
        5)
            echo "退出"
            exit 0
            ;;
        *)
            echo -e "${YELLOW}无效选项，请重新选择${NC}"
            ;;
    esac
    
    echo ""
    read -p "按 Enter 继续..."
    clear
done
