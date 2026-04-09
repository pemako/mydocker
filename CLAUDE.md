# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
make build           # 编译当前平台二进制
make build-linux     # 交叉编译 Linux amd64（无 CGO，exec 命令不可用）
make build-linux-cgo # 交叉编译 Linux amd64（含 CGO，exec 命令可用）
make check           # fmt-check + go vet（CI 首选）
make fmt             # 格式化代码
make tidy            # 整理 go.mod / go.sum
make test            # 在特权容器中构建并测试（自动选 docker / podman）
make clean           # 删除编译产物

# 直接 go 命令
go build -o mydocker .
sudo ./mydocker run -ti busybox sh   # 需要 Linux + root
```

No test files exist in the codebase. `go build ./...` is the primary correctness check.

## Architecture

All container operations require Linux; platform-specific files use `//go:build linux` / `//go:build !linux` build tags — every Linux-only file has a paired `_stub.go`.

### Container startup — two-process bootstrap

The `run` command forks a child via `os.Exec("/proc/self/exe", "init")`. The **parent** (`cmd/run.go: Run()`) calls `container.NewParentProcess()` which:
1. Creates an anonymous pipe for command delivery
2. Calls `container.NewWorkSpace()` to build the OverlayFS mount (lower/upper/work/merged under `/var/lib/mydocker/overlay2/<id>/`)
3. Starts the child with `CLONE_NEW{UTS,PID,NS,NET,IPC}` clone flags

The **child** runs `RunContainerInitProcess()` (`container/init_linux.go`) which:
1. Reads the command string from fd 3 (the read-end of the pipe)
2. Calls `setUpMount()` — bind-mounts root, `pivot_root`, mounts `/proc` and `/dev`
3. `syscall.Exec`s the user command, replacing itself

### exec — CGO namespace re-entry

`cmd/exec.go` sets `mydocker_pid` and `mydocker_cmd` env vars then re-executes `/proc/self/exe exec`. The CGO `__attribute__((constructor))` function in `nsenter/nsenter.go` runs *before* the Go runtime on every exec; when it detects `mydocker_pid` it calls `setns(2)` for all five namespaces and `execvp`s the command. The Go `ExecContainer` function path is a no-op in this case.

### Cgroup manager — factory pattern

`cgroups.NewCgroupManager(path)` calls `IsCgroup2UnifiedMode()` (detects via `statfs` magic `0x63677270`) and returns either a `CgroupManagerV1` (using `cgroups/subsystems/`) or `CgroupManagerV2` (using `cgroups/fs2/`). Both implement the same `CgroupManager` interface. v1 writes to `tasks`; v2 writes to `cgroup.procs`.

### Network — Linux-only split

`network/network.go` contains platform-agnostic code (structs, persistence, CRUD). Linux-only logic lives in:
- `network/bridge_linux.go` — `BridgeNetworkDriver`: creates bridge/veth via `netlink`, configures iptables MASQUERADE
- `network/connect_linux.go` — `connectImpl`: allocates IP, calls driver, enters container net ns (`enterContainerNetNS`), configures IP/route, sets up iptables DNAT port mapping

Non-Linux stubs in `bridge_stub.go` / `connect_stub.go` satisfy the compiler.

### Data paths at runtime

| What | Where |
|------|-------|
| Container metadata | `/var/run/mydocker/<name>/config.json` |
| Container log (detached) | `/var/run/mydocker/<name>/container.log` |
| Images | `/var/lib/mydocker/image/<name>.tar` |
| OverlayFS layers | `/var/lib/mydocker/overlay2/<id>/{lower,upper,work,merged}` |
| Network configs | `/var/lib/mydocker/network/network/<name>` |
| IPAM state | `/var/lib/mydocker/network/ipam/subnet.json` |

### Key design notes

- `container.RecordContainerInfo` returns `*ContainerInfo` (not a name string) — callers in `cmd/run.go` use it for the subsequent `network.Connect` call.
- `network.Connect` / `Disconnect` take `*container.ContainerInfo`; they load networks from disk on every call (no in-memory cache).
- `container.RemoveContainer(name, force bool)` — `force=true` calls `StopContainer` first; network cleanup is done by the caller (`cmd/rm.go`) to avoid a circular import.
- Log output is JSON-formatted (`logrus.JSONFormatter`); all user-visible output goes to `os.Stdout` via `fmt`, not via the logger.
