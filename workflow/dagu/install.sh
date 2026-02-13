#!/bin/bash

# Dagu 快速安装脚本

set -e

echo "=========================================="
echo "  Dagu 工作流引擎安装"
echo "=========================================="
echo ""

# 检测操作系统
OS="$(uname -s)"
ARCH="$(uname -m)"

echo "检测到系统: $OS $ARCH"
echo ""

# 安装 Dagu
if [ "$OS" = "Darwin" ]; then
    # macOS
    echo "在 macOS 上安装..."
    
    if command -v brew &> /dev/null; then
        echo "使用 Homebrew 安装 Dagu..."
        brew install dagu-org/brew/dagu
    else
        echo "❌ 未找到 Homebrew"
        echo "请先安装 Homebrew: https://brew.sh"
        echo "或手动下载: https://github.com/dagu-org/dagu/releases"
        exit 1
    fi
    
elif [ "$OS" = "Linux" ]; then
    # Linux
    echo "在 Linux 上安装..."
    
    if command -v go &> /dev/null; then
        echo "使用 Go 安装 Dagu..."
        go install github.com/dagu-org/dagu@latest
    else
        echo "❌ 未找到 Go"
        echo "请先安装 Go 或手动下载二进制文件"
        echo "下载地址: https://github.com/dagu-org/dagu/releases"
        exit 1
    fi
else
    echo "❌ 不支持的操作系统: $OS"
    exit 1
fi

echo ""
echo "=========================================="
echo "  安装完成！"
echo "=========================================="
echo ""

# 创建示例工作流目录
DAGU_DIR="$HOME/.dagu/dags"
mkdir -p "$DAGU_DIR"

echo "创建示例工作流..."

# 复制示例文件
if [ -d "examples" ]; then
    cp examples/*.yaml "$DAGU_DIR/"
    echo "✅ 示例工作流已复制到 $DAGU_DIR"
fi

echo ""
echo "下一步："
echo "1. 启动 Dagu:"
echo "   dagu start-all"
echo ""
echo "2. 访问 Web UI:"
echo "   http://localhost:8080"
echo ""
echo "3. 查看示例工作流:"
echo "   ls $DAGU_DIR"
echo ""
