BINARY    := mydocker
MODULE    := github.com/pemako/mydocker
GO        := go
GOFLAGS   :=

# 版本信息（从 git tag 读取，回退到 dev）
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS   := -X main.Version=$(VERSION) \
             -X main.Commit=$(COMMIT) \
             -X main.BuildTime=$(BUILD_TIME)

# Linux 交叉编译目标
LINUX_BINARY := $(BINARY)-linux-amd64

.DEFAULT_GOAL := build

# ── 构建 ──────────────────────────────────────────────────────────

.PHONY: build
build: ## 编译当前平台二进制（输出 ./mydocker）
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .

.PHONY: build-linux
build-linux: ## 交叉编译 Linux amd64 二进制（不含 CGO）
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(LINUX_BINARY) .

.PHONY: build-linux-cgo
build-linux-cgo: ## 交叉编译 Linux amd64 二进制（含 CGO，exec 命令需要）
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(LINUX_BINARY) .

# ── 代码质量 ──────────────────────────────────────────────────────

.PHONY: vet
vet: ## 运行 go vet
	$(GO) vet ./...

.PHONY: fmt
fmt: ## 格式化代码（gofmt -w）
	$(GO) fmt ./...

.PHONY: fmt-check
fmt-check: ## 检查代码格式（不修改文件，CI 用）
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "以下文件需要格式化："; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: lint
lint: ## 运行 golangci-lint（需已安装）
	golangci-lint run ./...

.PHONY: check
check: fmt-check vet ## 快速检查：格式 + vet（不需要额外工具）

# ── 依赖管理 ──────────────────────────────────────────────────────

.PHONY: tidy
tidy: ## 整理 go.mod / go.sum
	$(GO) mod tidy

.PHONY: download
download: ## 下载依赖
	$(GO) mod download

# ── 测试 ──────────────────────────────────────────────────────────

.PHONY: test
test: ## 在特权容器中构建并测试（需要 docker 或 podman）
	bash scripts/test-in-container.sh

# ── 数据目录初始化（Linux / root） ────────────────────────────────

.PHONY: init-dirs
init-dirs: ## 创建运行时所需目录（需要 sudo）
	sudo mkdir -p \
		/var/run/mydocker \
		/var/lib/mydocker/image \
		/var/lib/mydocker/overlay2 \
		/var/lib/mydocker/network/network \
		/var/lib/mydocker/network/ipam

# ── 清理 ──────────────────────────────────────────────────────────

.PHONY: clean
clean: ## 删除编译产物
	rm -f $(BINARY) $(LINUX_BINARY)

.PHONY: clean-data
clean-data: ## 删除运行时数据目录（需要 sudo，危险！）
	@echo "警告：将删除所有容器、网络、镜像数据。"
	@read -p "确认继续？[y/N] " ans && [ "$$ans" = "y" ]
	sudo rm -rf \
		/var/run/mydocker \
		/var/lib/mydocker/overlay2 \
		/var/lib/mydocker/network

# ── 帮助 ──────────────────────────────────────────────────────────

.PHONY: help
help: ## 显示此帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
