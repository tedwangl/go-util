#!/bin/bash

# Temporal 简单测试脚本

set -e

echo "=========================================="
echo "  Temporal 简单测试"
echo "=========================================="
echo ""

# 检查服务是否运行
echo "1. 检查 Temporal 服务状态..."
if ! curl -s http://localhost:7233 > /dev/null 2>&1; then
    echo "❌ Temporal 服务未运行"
    echo "请先启动服务: docker-compose up -d"
    exit 1
fi
echo "✅ Temporal 服务正在运行"
echo ""

# 检查 UI
echo "2. 检查 Temporal UI..."
if curl -s http://localhost:8088 > /dev/null 2>&1; then
    echo "✅ Temporal UI 可访问: http://localhost:8088"
else
    echo "⚠️  Temporal UI 暂时不可用，但服务正常"
fi
echo ""

echo "=========================================="
echo "  测试完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 访问 Web UI: http://localhost:8088"
echo "2. 启动 Worker: go run worker/main.go"
echo "3. 运行工作流: go run starter/main.go"
echo ""
