# MyDocker

一个简化的 Docker 容器运行时实现，用于学习和理解容器技术的核心原理。

## 项目简介

本项目是《自己动手写 Docker》的重构版本，使用 Go 1.25.5 重新实现，展示了容器技术的核心组件：

- **Linux Namespace**：实现进程、网络、文件系统等资源的隔离
- **Cgroups**：限制和控制容器的资源使用（CPU、内存等）
- **Container Networking**：实现容器网络连接和管理
- **Cobra CLI**：使用业界标准的 Cobra 框架构建命令行界面

## 技术特性

### 1. Namespace 隔离
- `CLONE_NEWUTS`：主机名和域名隔离
- `CLONE_NEWPID`：进程 ID 隔离
- `CLONE_NEWNS`：文件系统挂载点隔离
- `CLONE_NEWNET`：网络栈隔离
- `CLONE_NEWIPC`：进程间通信隔离

### 2. Cgroups 资源限制
- **内存限制**：限制容器可使用的最大内存
- **CPU 份额**：控制 CPU 使用权重
- **CPU 核心绑定**：将容器绑定到特定 CPU 核心

### 3. 容器网络
- **Bridge 网络**：基于 Linux Bridge 的容器网络
- **IPAM**：IP 地址自动分配和管理
- **Veth Pair**：容器和宿主机之间的虚拟网卡对
- **NAT**：容器访问外网的网络地址转换

## 系统要求

- **操作系统**：Linux（容器功能需要 Linux 内核支持）
- **Go 版本**：1.25.5 或更高
- **权限**：需要 root 权限运行（用于创建 namespace 和 cgroups）

⚠️ **重要**：由于容器技术依赖 Linux 内核特性，本项目只能在 Linux 系统上运行。如果在 macOS 或 Windows 上开发，需要使用虚拟机或容器环境。

## 安装

### 在 Linux 上直接构建

```bash
git clone https://github.com/xianlubird/mydocker.git
cd mydocker
go mod tidy
go build -o mydocker .
```

### 在 macOS/Windows 上交叉编译

```bash
# 克隆项目
git clone https://github.com/xianlubird/mydocker.git
cd mydocker
go mod tidy

# 交叉编译为 Linux 二进制
GOOS=linux GOARCH=amd64 go build -o mydocker .

# 将二进制文件传输到 Linux 机器
scp mydocker user@linux-host:/path/to/destination/
```

### 使用 Docker 测试（推荐用于非 Linux 系统）

```bash
# 在 Linux 容器中构建和测试
docker run --rm -it \
  --privileged \
  -v $(pwd):/workspace \
  -w /workspace \
  golang:1.25.5 \
  bash

# 容器内执行
go mod tidy
go build -o mydocker .
# 注意：需要 --privileged 才能使用 namespace 和 cgroups
```

## 使用方法

### 容器运行命令

```bash
# 以交互模式运行容器
sudo ./mydocker run -ti busybox sh

# 以后台模式运行容器
sudo ./mydocker run -d --name mycontainer busybox sh

# 运行容器并限制资源
sudo ./mydocker run -ti -m 100m --cpushare 512 --cpuset 0-1 busybox sh

# 运行容器并挂载卷
sudo ./mydocker run -ti -v /host/path:/container/path busybox sh

# 运行容器并设置环境变量
sudo ./mydocker run -ti -e ENV1=value1 -e ENV2=value2 busybox sh

# 运行容器并配置端口映射
sudo ./mydocker run -ti -p 8080:80 busybox sh

# 组合多个选项
sudo ./mydocker run -d --name web -m 100m -v /data:/app -e APP_ENV=prod -p 8080:80 busybox sh
```

### 容器管理命令

```bash
# 列出所有容器
sudo ./mydocker ps

# 查看容器日志
sudo ./mydocker logs mycontainer

# 在运行的容器中执行命令
sudo ./mydocker exec mycontainer sh

# 停止容器
sudo ./mydocker stop mycontainer

# 删除已停止的容器
sudo ./mydocker rm mycontainer

# 将容器提交为镜像
sudo ./mydocker commit mycontainer myimage
```

### 网络管理命令

```bash
# 创建网络
sudo ./mydocker network create --driver bridge --subnet 192.168.0.0/24 mynet

# 列出所有网络
sudo ./mydocker network list

# 删除网络
sudo ./mydocker network remove mynet
```

### run 命令参数

- `-ti, -t`：启用交互式终端（TTY）
- `-d`：后台模式运行容器
- `--name`：指定容器名称
- `-m`：内存限制（例如：100m, 1g）
- `--cpushare`：CPU 份额权重（默认 1024）
- `--cpuset`：绑定的 CPU 核心（例如：0, 0-2, 0,2,4）
- `-v`：卷挂载（格式：宿主机路径:容器路径）
- `-e`：设置环境变量（可多次使用）
- `--net`：指定容器网络
- `-p`：端口映射（格式：宿主机端口:容器端口，可多次使用）
```

## 项目架构

```
mydocker/
├── main.go                          # 程序入口（24 行）
├── cmd/
│   ├── root.go                      # 根命令定义
│   ├── run.go                       # 运行容器命令
│   ├── init.go                      # 初始化命令
│   └── network.go                   # 网络管理命令
├── container/
│   ├── container_process.go         # 通用代码
│   ├── container_process_linux.go   # Linux 特定的进程创建
│   ├── container_process_stub.go    # 非 Linux 平台 stub
│   ├── init_linux.go                # 容器初始化进程
│   └── init_stub.go                 # 非 Linux 平台 stub
├── cgroups/
│   ├── cgroup_manager.go            # Cgroups 管理器
│   └── subsystems/
│       ├── subsystem.go             # 子系统接口定义
│       ├── memory.go                # 内存子系统
│       ├── cpu.go                   # CPU 份额子系统
│       ├── cpuset.go                # CPU 核心绑定子系统
│       └── utils.go                 # 工具函数
├── network/
│   ├── network.go                   # 网络核心接口
│   ├── ipam.go                      # IP 地址管理
│   ├── bridge_linux.go              # Bridge 网络驱动（Linux）
│   └── bridge_stub.go               # Bridge stub（非 Linux）
├── go.mod
└── README.md
```

### 代码组织

- **main.go**：程序唯一入口，负责初始化 CLI 应用（仅 24 行代码）
- **cmd/**：命令包，每个命令独立文件，职责清晰
  - `root.go`：根命令和全局配置
  - `run.go`：容器运行逻辑（77 行）
  - `init.go`：容器初始化（19 行）
  - `network.go`：网络管理（53 行）
- **container/**：容器进程管理，包括 namespace 隔离
- **cgroups/**：资源限制管理，实现 CPU、内存等资源控制
- **network/**：容器网络管理，包括 Bridge 驱动和 IPAM

## 核心实现

### 1. 进程隔离
使用 `syscall.SysProcAttr` 的 `Cloneflags` 创建新的 namespace：

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | 
                syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | 
                syscall.CLONE_NEWIPC,
}
```

### 2. 资源限制
通过写入 cgroup 文件系统实现资源控制：

```go
// 限制内存
ioutil.WriteFile("/sys/fs/cgroup/memory/mydocker-cgroup/memory.limit_in_bytes", 
                  []byte("100m"), 0644)

// 限制 CPU 份额
ioutil.WriteFile("/sys/fs/cgroup/cpu/mydocker-cgroup/cpu.shares", 
                  []byte("512"), 0644)
```

### 3. 进程通信
使用匿名管道在父子进程间传递命令：

```go
readPipe, writePipe, _ := os.Pipe()
cmd.ExtraFiles = []*os.File{readPipe}
// 父进程通过 writePipe 发送命令
// 子进程从文件描述符 3 读取命令
```

## 功能特性

### 已实现功能

- [x] **基本容器运行**：支持交互式和后台运行模式
- [x] **Namespace 隔离**：实现 5 种 namespace 隔离
- [x] **Cgroups 资源限制**：支持内存、CPU 份额、CPU 核心绑定
- [x] **容器管理**：ps、logs、stop、rm 等完整生命周期管理
- [x] **容器执行**：exec 命令在运行的容器中执行命令（使用 nsenter）
- [x] **镜像管理**：commit 命令将容器提交为镜像
- [x] **卷挂载**：支持宿主机目录挂载到容器（基于 AUFS）
- [x] **环境变量**：支持设置容器环境变量
- [x] **网络管理**：创建、列出、删除容器网络
- [x] **端口映射**：支持端口映射配置
- [x] **容器命名**：支持自定义容器名称
- [x] **Cobra CLI**：使用业界标准框架，提供清晰的命令结构

### 待实现功能

- [ ] AUFS/OverlayFS 文件系统完整实现
- [ ] 容器网络连接和通信
- [ ] 完整的镜像管理（镜像存储、导入导出）
- [ ] 容器日志轮转
- [ ] 容器资源统计
- [ ] 安全加固（seccomp、capability 等）

## 注意事项

⚠️ **本项目仅用于学习目的，不适用于生产环境！**

- 需要在 Linux 系统上运行
- 需要 root 权限
- 不包含安全加固措施
- 功能简化，未实现完整的容器特性

## 参考资料

- 《自己动手写 Docker》
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [Linux Control Groups](https://www.kernel.org/doc/Documentation/cgroup-v1/cgroups.txt)

## 许可证

本项目基于原始项目继承相应许可证。

## 贡献

欢迎提交 Issue 和 Pull Request！
