# mydocker

一个用于学习容器技术核心原理的简易容器运行时，参考《自己动手写 Docker》使用 Go 重新实现。

## 功能特性

| 功能                  | 说明                                                      |
| --------------------- | --------------------------------------------------------- |
| Namespace 隔离        | UTS / PID / Mount / Network / IPC 五种隔离                |
| Cgroup v1/v2 资源限制 | 内存、CPU 份额、CPU 核心绑定，自动检测内核版本            |
| OverlayFS 文件系统    | lower / upper / work / merged 四层结构，替代已废弃的 AUFS |
| 容器网络              | Linux Bridge + veth pair + iptables SNAT/DNAT             |
| IPAM                  | 基于位图的子网 IP 分配，持久化到磁盘                      |
| 网络持久化            | 网络配置重启后自动恢复                                    |
| 端口映射              | iptables PREROUTING DNAT 规则                             |
| exec 进入容器         | CGO + setns 系统调用，进入容器所有 Namespace              |
| 容器镜像提交          | 将容器 merged 层打包为新的镜像 tar 包                     |
| 数据卷挂载            | bind mount 宿主机目录到容器内                             |
| 容器重启              | 使用保存的配置（镜像、命令、环境变量）重新启动            |
| 容器详情              | JSON 格式输出完整容器元数据                               |

## 系统要求

- **操作系统**：Linux（容器功能强依赖 Linux 内核）
- **Go 版本**：1.21+
- **内核版本**：建议 5.4+（OverlayFS、cgroup v2）
- **权限**：需要 root 运行（Namespace、cgroup、iptables 操作）
- **依赖工具**：`iptables`、`tar`、`mount`

> 在 macOS / Windows 上开发时，可交叉编译后传到 Linux 机器测试，
> 或使用 `--privileged` Docker 容器作为测试环境（见[开发环境](#开发环境)）。

## 快速开始

### 编译

```bash
git clone https://github.com/pemako/mydocker.git
cd mydocker
go build -o mydocker .
```

交叉编译（macOS → Linux）：

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc \
  go build -o mydocker .
```

### 准备镜像

mydocker 使用 tar 包作为镜像格式，默认存放在 `/var/lib/mydocker/image/`：

```bash
# 导出 busybox 根文件系统
docker export $(docker create busybox) -o busybox.tar
sudo mkdir -p /var/lib/mydocker/image/
sudo mv busybox.tar /var/lib/mydocker/image/busybox.tar
```

### 运行第一个容器

```bash
# 交互式运行
sudo ./mydocker run -ti busybox sh

# 后台运行
sudo ./mydocker run -d --name demo busybox top

# 查看容器列表
sudo ./mydocker ps
```

## 命令参考

### run — 创建并运行容器

```
sudo ./mydocker run [flags] IMAGE [COMMAND...]
```

| 参数         | 说明                                     | 示例             |
| ------------ | ---------------------------------------- | ---------------- |
| `-ti`        | 前台交互模式（分配 TTY）                 | `-ti`            |
| `-d`         | 后台运行                                 | `-d`             |
| `--name`     | 容器名称（默认随机 ID）                  | `--name web`     |
| `-m`         | 内存限制                                 | `-m 256m`        |
| `--cpushare` | CPU 权重（cgroup v1，默认 1024）         | `--cpushare 512` |
| `--cpuset`   | 绑定 CPU 核心                            | `--cpuset 0-1`   |
| `-v`         | 数据卷挂载 `宿主机路径:容器路径`         | `-v /data:/app`  |
| `-e`         | 环境变量（可重复）                       | `-e KEY=val`     |
| `--net`      | 连接到指定网络                           | `--net mynet`    |
| `-p`         | 端口映射 `宿主机端口:容器端口`（可重复） | `-p 8080:80`     |

```bash
# 限制资源
sudo ./mydocker run -ti -m 128m --cpushare 512 --cpuset 0 busybox sh

# 挂载数据卷 + 环境变量
sudo ./mydocker run -d --name app -v /host/data:/data -e APP_ENV=prod busybox top

# 接入网络 + 端口映射
sudo ./mydocker run -d --name web --net mynet -p 8080:80 busybox httpd -f -p 80
```

### 容器生命周期

```bash
# 查看所有容器
sudo ./mydocker ps

# 查看后台容器日志
sudo ./mydocker logs <name>

# 在运行中的容器内执行命令
sudo ./mydocker exec <name> sh

# 停止容器（发送 SIGTERM）
sudo ./mydocker stop <name>

# 重启容器
sudo ./mydocker restart <name>

# 删除已停止的容器
sudo ./mydocker rm <name>

# 强制删除运行中的容器
sudo ./mydocker rm -f <name>

# 查看容器详细信息（JSON）
sudo ./mydocker inspect <name>

# 将容器提交为新镜像
sudo ./mydocker commit <name> <image-name>
```

### 网络管理

```bash
# 创建 bridge 网络
sudo ./mydocker network create --driver bridge --subnet 172.18.0.0/24 mynet

# 查看所有网络
sudo ./mydocker network list

# 删除网络
sudo ./mydocker network remove mynet
```

## 项目结构

```
mydocker/
├── main.go                           # 程序入口
├── doc.go                            # 包文档
│
├── cmd/                              # CLI 子命令（cobra）
│   ├── root.go                       # 根命令 & 子命令注册
│   ├── run.go                        # run：创建并运行容器
│   ├── init.go                       # init：容器内部初始化（内部命令）
│   ├── stop.go                       # stop：停止容器
│   ├── rm.go                         # rm：删除容器（支持 -f）
│   ├── ps.go                         # ps：列出容器
│   ├── exec.go                       # exec：进入容器执行命令
│   ├── logs.go                       # logs：查看容器日志
│   ├── inspect.go                    # inspect：查看容器详情
│   ├── restart.go                    # restart：重启容器
│   ├── commit.go                     # commit：提交容器为镜像
│   └── network.go                    # network：网络子命令组
│
├── container/                        # 容器核心逻辑
│   ├── container_info.go             # 容器元数据 CRUD、生命周期管理
│   ├── container_process_linux.go    # 父进程创建（Namespace clone、OverlayFS）
│   ├── container_process_stub.go     # 非 Linux 平台 stub
│   ├── init_linux.go                 # 子进程初始化（mount、pivotRoot、exec）
│   ├── init_stub.go                  # 非 Linux 平台 stub
│   ├── volume.go                     # OverlayFS 文件系统构建与清理
│   └── utils.go                      # 工具函数（PathExists、KillProcess 等）
│
├── cgroups/                          # Cgroup 资源限制
│   ├── cgroup_manager.go             # CgroupManager 接口 + 工厂函数（v1/v2 自动选择）
│   ├── cgroup_manager_v1.go          # Cgroup v1 实现
│   ├── cgroup_manager_v2.go          # Cgroup v2 实现
│   ├── util.go                       # IsCgroup2UnifiedMode 检测
│   ├── util_stub.go                  # 非 Linux 平台 stub
│   ├── subsystems/                   # Cgroup v1 子系统
│   │   ├── subsystem.go              # Subsystem 接口 + ResourceConfig
│   │   ├── memory.go                 # memory.limit_in_bytes
│   │   ├── cpu.go                    # cpu.shares
│   │   ├── cpuset.go                 # cpuset.cpus
│   │   └── utils.go                  # 挂载点查找（/proc/self/mountinfo）
│   └── fs2/                          # Cgroup v2 子系统
│       ├── subsystems.go             # 子系统列表
│       ├── defaultpath.go            # UnifiedMountpoint
│       ├── utils.go                  # getCgroupPath + applyCgroup（cgroup.procs）
│       ├── memory.go                 # memory.max
│       ├── cpu.go                    # cpu.max（不支持 cpu.shares）
│       └── cpuset.go                 # cpuset.cpus
│
├── network/                          # 容器网络
│   ├── network.go                    # Network/Endpoint/Driver 定义 + CRUD + 持久化
│   ├── ipam.go                       # IPAM 位图算法（分配/释放 IP）
│   ├── bridge_linux.go               # BridgeNetworkDriver（Linux）
│   ├── bridge_stub.go                # 非 Linux 平台 stub
│   ├── connect_linux.go              # enterContainerNetNS + IP 路由 + 端口映射
│   └── connect_stub.go               # 非 Linux 平台 stub
│
└── nsenter/                          # Namespace 进入（CGO）
    ├── nsenter.go                    # C 构造函数：在 Go runtime 前执行 setns
    └── nsenter_stub.go               # 非 Linux/CGO 平台 stub
```

## 核心原理

### 容器启动流程

```
mydocker run busybox sh
      │
      ▼
NewParentProcess()               ← 父进程（mydocker）
  clone(NEWUTS|NEWPID|NEWNS|NEWNET|NEWIPC)
  NewWorkSpace()                 ← 创建 OverlayFS 四层目录并挂载
  cmd.Dir = merged/              ← 设置工作目录为容器根
  parent.Start()                 ← fork 子进程
      │
      ▼
RunContainerInitProcess()        ← 子进程（mydocker init）
  readUserCommand()              ← 从管道读取命令（"sh"）
  setUpMount()                   ← pivot_root + 挂载 /proc /dev
  syscall.Exec("sh", ...)        ← 替换自身为用户进程
```

### OverlayFS 目录结构

```
/var/lib/mydocker/
├── image/
│   └── busybox.tar              # 镜像 tar 包
└── overlay2/
    └── <containerID>/
        ├── lower/               # 只读层（镜像解压）
        ├── upper/               # 读写层（容器内写操作）
        ├── work/                # OverlayFS 工作目录
        └── merged/              # 联合挂载点（容器根目录）

# 挂载命令
mount -t overlay overlay \
  -o lowerdir=lower,upperdir=upper,workdir=work \
  merged
```

### 容器网络架构

```
宿主机                              容器
──────────────────────────────────────────────────
                    veth pair
  Bridge (mynet) ←──────────── eth0 (cif-xxxxx)
  172.18.0.1                    172.18.0.2/24

iptables:
  POSTROUTING MASQUERADE   ← 容器访问外网（SNAT）
  PREROUTING DNAT 8080→80  ← 端口映射（-p 8080:80）
```

### exec 进入容器原理

```
mydocker exec <name> sh
      │
      ├─ 读取容器 PID，设置环境变量 mydocker_pid=<pid>
      │
      ▼
/proc/self/exe exec           ← 重新执行自身
      │
      ▼  (CGO __attribute__((constructor)) 在 Go runtime 前触发)
enter_namespace()             ← C 函数
  setns(/proc/<pid>/ns/mnt)
  setns(/proc/<pid>/ns/net)
  setns(/proc/<pid>/ns/uts)
  setns(/proc/<pid>/ns/ipc)
  setns(/proc/<pid>/ns/pid)
  execvp("sh")
```

## 数据目录

| 路径                                         | 说明                           |
| -------------------------------------------- | ------------------------------ |
| `/var/run/mydocker/<name>/config.json`       | 容器元数据（PID、状态、IP 等） |
| `/var/run/mydocker/<name>/container.log`     | 后台容器标准输出日志           |
| `/var/lib/mydocker/image/<name>.tar`         | 镜像 tar 包                    |
| `/var/lib/mydocker/overlay2/<id>/`           | 容器 OverlayFS 目录            |
| `/var/lib/mydocker/network/network/<name>`   | 网络配置（JSON）               |
| `/var/lib/mydocker/network/ipam/subnet.json` | IPAM 分配状态                  |

## 开发环境

macOS / Windows 上可使用 Docker 特权容器作为测试环境：

```bash
docker run --rm -it --privileged \
  -v $(pwd):/workspace -w /workspace \
  golang:1.21 bash

# 容器内
go build -o mydocker .
# 准备镜像后即可测试
```

## 依赖

| 包                                                                       | 用途                   |
| ------------------------------------------------------------------------ | ---------------------- |
| [github.com/spf13/cobra](https://github.com/spf13/cobra)                 | CLI 框架               |
| [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)         | 结构化日志             |
| [github.com/vishvananda/netlink](https://github.com/vishvananda/netlink) | Linux 网络设备操作     |
| [github.com/vishvananda/netns](https://github.com/vishvananda/netns)     | Network Namespace 操作 |

## 延伸阅读

深入了解各核心技术的原理与实现细节，请参阅 [docs/docker-internals.md](docs/docker-internals.md)，内容涵盖：

- Namespace / Cgroups / OverlayFS 原理精讲
- 容器进程管道通信与 pivot_root 详解
- nsenter + CGO setns 进入 Namespace 的实现原理
- 从零实现 mini-Docker 的分步骤教程（Step 1–10）
- 本项目与真实 Docker 的技术对比

## 参考资料

- 《自己动手写 Docker》— 陈显鹭
- [Linux man-pages: namespaces(7)](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [Linux man-pages: cgroups(7)](https://man7.org/linux/man-pages/man7/cgroups.7.html)
- [Kernel docs: overlayfs](https://www.kernel.org/doc/html/latest/filesystems/overlayfs.html)
- [Kernel docs: cgroup-v2](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html)
- [lixd/mydocker](https://github.com/lixd/mydocker)
