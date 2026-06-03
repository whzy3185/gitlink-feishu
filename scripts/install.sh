#!/bin/bash
# GitLink CLI 一键安装脚本 (Linux / macOS)
set -e

echo "========================================="
echo "  GitLink CLI 安装脚本"
echo "========================================="

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 转换架构名称
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l)  ARCH="armv7" ;;
    *)       echo "不支持的架构: $ARCH"; exit 1 ;;
esac

echo "检测到系统: ${OS} ${ARCH}"

BINARY="gitlink-cli-${OS}-${ARCH}"
URL="https://gitlink.org.cn/Gitlink/gitlink-cli/releases/download/latest/${BINARY}"

echo "下载地址: $URL"

# 下载
if command -v curl &> /dev/null; then
    curl -fsSL "$URL" -o /tmp/gitlink-cli
elif command -v wget &> /dev/null; then
    wget -q "$URL" -O /tmp/gitlink-cli
else
    echo "错误: 需要 curl 或 wget"
    exit 1
fi

# 安装
chmod +x /tmp/gitlink-cli

if [ "$(id -u)" -eq 0 ]; then
    mv /tmp/gitlink-cli /usr/local/bin/gitlink-cli
else
    echo "需要 sudo 权限安装到 /usr/local/bin/"
    sudo mv /tmp/gitlink-cli /usr/local/bin/gitlink-cli
fi

echo ""
echo "✅ 安装完成！"
echo "   运行 gitlink-cli --help 验证安装"
