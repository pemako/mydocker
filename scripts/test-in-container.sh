#!/usr/bin/env bash
# scripts/test-in-container.sh
# 在特权容器中构建并测试 mydocker（适用于 macOS / Windows 开发机）
#
# 优先使用 docker；若未安装则自动回退到 podman。
# 环境变量（均可从外部覆盖）：
#   GO_IMAGE — 使用的 Go 镜像，默认 golang:1.25.5
#   BINARY   — 编译输出的二进制名，默认 mydocker
#   GOPROXY  — Go 模块代理，默认 http://goproxy.cn,direct

set -euo pipefail

# ── 检测容器运行时 ────────────────────────────────────────────────
detect_runtime() {
    if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
        echo "docker"
    elif command -v podman &>/dev/null; then
        echo "podman"
    else
        echo ""
    fi
}

RUNTIME=$(detect_runtime)

if [[ -z "$RUNTIME" ]]; then
    echo "错误：未找到可用的容器运行时（docker / podman），请先安装其中一个。" >&2
    exit 1
fi

# ── 参数 ──────────────────────────────────────────────────────────
GO_IMAGE="${GO_IMAGE:-golang:1.25.5}"
BINARY="${BINARY:-mydocker}"
GOPROXY="${GOPROXY:-http://goproxy.cn,direct}"
WORKSPACE="/workspace"

echo "容器运行时：$RUNTIME"
echo "Go 镜像：$GO_IMAGE"
echo "GOPROXY：$GOPROXY"

# ── 启动特权容器 ──────────────────────────────────────────────────
# 变量通过 -e 注入容器，bash -c 使用单引号避免外层 shell 转义问题
echo "启动容器（--privileged）..."

"$RUNTIME" run --rm -it \
    --privileged \
    -v "$(pwd):$WORKSPACE" \
    -w "$WORKSPACE" \
    -e "GOPROXY=$GOPROXY" \
    -e "BINARY=$BINARY" \
    "$GO_IMAGE" \
    bash -c '
set -euo pipefail

echo "==> 设置 GOPROXY: $GOPROXY"
go env -w GOPROXY="$GOPROXY"

echo "==> 安装/整理依赖..."
go mod tidy

echo "==> 编译 $BINARY ..."
go build -o "$BINARY" .

echo "==> 编译成功"
ls -lh "$BINARY"

echo ""
echo "可用命令示例："
echo "  准备镜像："
echo "    docker export \$(docker create busybox) -o /tmp/busybox.tar"
echo "    mkdir -p /var/lib/mydocker/image"
echo "    cp /tmp/busybox.tar /var/lib/mydocker/image/busybox.tar"
echo ""
echo "  运行容器："
echo "    sudo ./$BINARY run -ti busybox sh"
echo "    sudo ./$BINARY run -d --name demo busybox top"
echo "    sudo ./$BINARY ps"
echo ""
echo "提示：编译验证已完成。"
echo "      容器内套容器环境受限（无 docker daemon、OverlayFS 受限），"
echo "      完整功能测试（run/network/exec）需在 Linux 物理机或虚拟机上以 root 执行。"
echo ""

exec bash -l
'
