#!/bin/bash
# 在 Docker 容器中测试 mydocker
# 此脚本适用于 macOS/Windows 用户

set -e

echo "🐳 启动 Docker 容器进行测试..."
docker run --rm -it \
  --privileged \
  -v "$(pwd):/workspace" \
  -w /workspace \
  golang:1.25.5 \
  bash -c '
echo "📦 安装依赖..."
go mod tidy

echo "🔨 编译项目..."
go build -o mydocker .

echo "✅ 编译成功！文件信息："
ls -lh mydocker
file mydocker

echo ""
echo "🚀 现在你可以运行以下命令测试 mydocker："
echo "   sudo ./mydocker run -ti /bin/sh"
echo ""
echo "💡 注意：你现在在一个特权容器中，可以测试 namespace 和 cgroups"
echo ""

# 启动交互式 shell
exec bash
'
