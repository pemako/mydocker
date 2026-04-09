# Docker 技术原理深度解析与实现指南

## Docker 的六大核心技术原理

### 2.1 Linux Namespace — 隔离

**Namespace 是容器隔离的基础**，让容器内的进程感知不到宿主机和其他容器的存在。Linux 提供 7 种 Namespace：

| Namespace | 系统调用 flag     | 隔离内容                     |
| --------- | ----------------- | ---------------------------- |
| UTS       | `CLONE_NEWUTS`    | hostname、domainname         |
| PID       | `CLONE_NEWPID`    | 进程 ID 空间（容器内 PID 1） |
| Mount     | `CLONE_NEWNS`     | 挂载点视图                   |
| Network   | `CLONE_NEWNET`    | 网络设备、路由、iptables     |
| IPC       | `CLONE_NEWIPC`    | System V IPC、POSIX 消息队列 |
| User      | `CLONE_NEWUSER`   | UID/GID 映射                 |
| Cgroup    | `CLONE_NEWCGROUP` | cgroup 根目录（较新内核）    |

**本项目实现**（`container/container_process_linux.go:29-32`）：

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
}
```

调用 `clone()` 系统调用时传入这些 flag，内核会为新进程创建独立的 Namespace 视图。

**原理深入**：

- 创建进程时内核复制父进程的 Namespace 引用，传入 `CLONE_NEW*` 时则新建
- `/proc/<pid>/ns/` 目录下可看到各 Namespace 的 inode，相同 inode = 同一 Namespace
- `nsenter` 命令（及系统调用 `setns()`）可以让进程加入已存在的 Namespace

---

### 2.2 Linux Cgroups — 资源限制

**Cgroups（Control Groups）** 控制进程能使用多少资源，防止"噪音邻居"问题。

**架构**：

```
/sys/fs/cgroup/
├── memory/          ← memory 子系统挂载点
│   └── mydocker-cgroup/
│       ├── memory.limit_in_bytes   ← 写入内存上限
│       └── tasks                   ← 写入 PID 即加入该 cgroup
├── cpu/
│   └── mydocker-cgroup/
│       └── cpu.shares
└── cpuset/
    └── mydocker-cgroup/
        └── cpuset.cpus
```

**本项目实现流程**（`cgroups/`）：

1. `FindCgroupMountpoint()` — 读 `/proc/self/mountinfo` 找到子系统的挂载路径
2. `GetCgroupPath()` — 在挂载点下创建 cgroup 子目录（mkdir 即创建 cgroup）
3. `Set()` — 向 `memory.limit_in_bytes` 等控制文件写入限制值
4. `Apply()` — 将容器 PID 写入 `tasks` 文件，使该进程受限

```go
// memory.go:17 — 设置内存限制的核心
ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.limit_in_bytes"),
    []byte(res.MemoryLimit), 0644)

// memory.go:38 — 将进程加入 cgroup
ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"),
    []byte(strconv.Itoa(pid)), 0644)
```

**注意**：Linux 内核 4.5+ 引入了 cgroups v2（unified hierarchy），现代发行版（Ubuntu 22.04+）默认使用 v2，路径结构有所不同。

---

### 2.3 Union Filesystem — 镜像分层

Union FS 允许将多个目录叠加为一个统一视图，是镜像分层的基础。

**本项目使用 AUFS**，实现写时复制（Copy-on-Write）：

```
/root/
├── ubuntu/          ← 镜像解压后的只读层（镜像层）
├── writeLayer/
│   └── <containerName>/  ← 容器读写层（每个容器独立）
└── mnt/
    └── <containerName>/  ← AUFS 联合挂载点（容器看到的文件系统）
```

**挂载命令**（`container/volume.go:86`）：

```bash
mount -t aufs -o dirs=<writeLayer>:<imageLayer> none <mntPoint>
```

读写规则：

- **读文件**：从上往下查找，writeLayer → imageLayer
- **写文件**：如果文件在只读层，先复制到 writeLayer（CoW），再修改
- **删文件**：在 writeLayer 创建 `.wh.filename` 白障文件遮盖只读层

**容器启动后文件系统切换**（`container/init_linux.go:65-95`，`pivotRoot`）：

1. `mount --bind <rootfs> <rootfs>` — 将 rootfs 绑定挂载到自身（为 pivot_root 做准备）
2. `pivot_root <new_root> <old_root>` — 切换根目录，旧根挂在 `.pivot_root/`
3. `umount <old_root>` — 卸载旧根，容器与宿主文件系统彻底隔离

**现代 Docker 默认使用 overlay2**，原理相同但性能更好：

```
lowerdir=image_layers（只读，可多层）
upperdir=container_layer（读写）
workdir=work（内核工作目录）
merged=merged（联合视图）
```

---

### 2.4 容器 Init 进程与管道通信

**设计亮点**：父进程（宿主机）和子进程（容器）通过匿名管道传递启动命令。

```
宿主机进程                       容器进程
─────────────────────────────────────────────
Run()                           RunContainerInitProcess()
  │                                   │
  ├─ NewPipe() 创建读写管道             │
  ├─ cmd.ExtraFiles = [readPipe]       │
  ├─ parent.Start() ──────────────────►│ fork/clone 进入新 Namespace
  │                                   │ 读取 fd[3]（readPipe）
  ├─ cgroupManager.Set/Apply()        │   阻塞等待命令...
  │                                   │
  ├─ sendInitCommand(comArray, writePipe) ──► 写入命令字符串
  │    writePipe.WriteString("sh")    │
  │    writePipe.Close()              │ 读到命令，关闭管道
  │                                   │ syscall.Exec("sh") 替换自身进程
```

关键点：

- 子进程继承了父进程的文件描述符（fd[3] = readPipe）
- `syscall.Exec()` 是 `execve` 系统调用，**用目标程序替换当前进程**（保留 PID、Namespace），这是 PID 1 的来源
- 管道关闭前子进程阻塞，确保 cgroup 设置完成后再启动用户程序

---

### 2.5 容器 exec — nsenter 进入 Namespace

**难点**：Go 的 runtime 在进程启动时会启动多个线程，而 `setns()` 必须在单线程时调用。解决方案是用 cgo 在 Go runtime 初始化前执行 C 代码。

**本项目实现**（`nsenter/` + `cmd/exec.go`）：

```
mydocker exec <container> <command>
        │
        ├─ 读取容器 PID
        ├─ 设置环境变量 mydocker_pid=<pid>, mydocker_cmd=<cmd>
        └─ exec.Command("/proc/self/exe", "exec")  ← 重新执行自身
                │
                └─ [C 代码 nsenter init]  ← 在 Go main() 之前执行
                        │  检测到 mydocker_pid 环境变量
                        └─ setns() 进入 PID/Mount/Net/IPC namespace
                                │
                                └─ execve(cmd)  执行用户命令
```

---

### 2.6 容器网络 — Bridge + Veth + iptables

**网络模型**（类似 docker0 默认网络）：

```
宿主机
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  eth0 (宿主机网卡)                                        │
│    │                                                     │
│  iptables MASQUERADE (SNAT)                              │
│    │                                                     │
│  br0 (Linux Bridge，网关 172.17.0.1)                     │
│  ├── veth0 (宿主机端)  ←──────────────── 容器A            │
│  │                               veth1 (容器端 172.17.0.2)│
│  └── veth2 (宿主机端)  ←──────────────── 容器B            │
│                                   veth3 (容器端 172.17.0.3)│
└──────────────────────────────────────────────────────────┘
```

**实现步骤**（`network/bridge_linux.go`）：

1. `netlink.LinkAdd(&Bridge{})` — 创建 Linux Bridge 设备
2. `netlink.AddrAdd()` — 给 Bridge 配置 IP（作为容器网关）
3. `netlink.LinkSetUp()` — 启动 Bridge
4. `iptables -t nat -A POSTROUTING -s <subnet> ! -o <bridge> -j MASQUERADE` — 允许容器访问外网
5. 容器连接时：创建 veth pair，一端加入 Bridge，另一端放入容器 Network Namespace，配置 IP

**IPAM** (`network/ipam.go`)：用位图（字符串模拟）管理子网内的 IP 分配，持久化到 `/var/run/mydocker/network/ipam/subnet.json`。

---

## 三、完整数据流：`mydocker run -ti ubuntu sh`

```
用户输入
    │
    ▼
cmd/run.go: Run()
    │
    ├─ 1. NewParentProcess()
    │      ├─ 创建匿名管道 (readPipe, writePipe)
    │      ├─ exec.Command("/proc/self/exe", "init")
    │      ├─ SysProcAttr.Cloneflags = UTS|PID|NS|NET|IPC
    │      ├─ NewWorkSpace()
    │      │    ├─ CreateReadOnlyLayer()  解压 ubuntu.tar → /root/ubuntu/
    │      │    ├─ CreateWriteLayer()     mkdir /root/writeLayer/<name>/
    │      │    └─ CreateMountPoint()     mount -t aufs → /root/mnt/<name>/
    │      └─ cmd.Dir = "/root/mnt/<name>"  (容器 rootfs)
    │
    ├─ 2. parent.Start()  → clone() 系统调用
    │      子进程在新 Namespace 中运行 "mydocker init"
    │      子进程阻塞在读管道 fd[3]
    │
    ├─ 3. RecordContainerInfo()  写 /var/run/mydocker/<name>/config.json
    │
    ├─ 4. cgroupManager.Set()    写 memory.limit_in_bytes 等
    │      cgroupManager.Apply() 写 tasks，将子进程 PID 加入 cgroup
    │
    ├─ 5. sendInitCommand(["sh"], writePipe)
    │      写入 "sh"，关闭 writePipe
    │
    └─ 6. parent.Wait()  (tty 模式等待容器退出)

子进程（容器）
    │
    ├─ RunContainerInitProcess()
    │      ├─ readUserCommand()  从 fd[3] 读到 "sh"
    │      ├─ (setUpMount → pivotRoot，本项目已注释)
    │      └─ syscall.Exec("/bin/sh", ["sh"], environ)
    │            替换当前进程为 sh，PID 不变
    │
    └─ 用户在容器内的 shell
```

---

## 四、从零实现一个学习用 mini-Docker：详细步骤

### 前置条件

- Linux 系统（Ubuntu 20.04+），推荐在虚拟机或 Linux 服务器上开发
- Go 1.21+
- 基本了解 Go 语言和 Linux 系统调用

### Step 1：搭建项目骨架

```bash
mkdir mydocker && cd mydocker
go mod init github.com/yourname/mydocker
go get github.com/spf13/cobra
go get github.com/sirupsen/logrus
```

用 cobra 创建 CLI 框架，实现 `run` 和 `init` 两个子命令。

**验证目标**：`go build` 通过，`./mydocker --help` 显示帮助。

---

### Step 2：实现 Namespace 隔离（最小可运行容器）

核心：用 `exec.Cmd` 的 `SysProcAttr.Cloneflags` 创建隔离进程。

```go
// 父进程：run.go
cmd := exec.Command("/proc/self/exe", "init")
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
}
cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

// 子进程：init.go（容器内第一个进程）
func RunContainerInitProcess() error {
    // 用 syscall.Exec 替换自身，保证 PID=1
    return syscall.Exec("/bin/sh", []string{"sh"}, os.Environ())
}
```

**验证目标**：

```bash
sudo ./mydocker run
# 容器内 hostname 与宿主机不同
# 容器内 echo $$ 显示 1
```

---

### Step 3：父子进程管道通信

目标：父进程通过管道告诉子进程运行什么命令。

```go
// 创建管道
readPipe, writePipe, _ := os.Pipe()
cmd.ExtraFiles = []*os.File{readPipe}  // 子进程 fd[3]

// 父进程：启动子进程后发命令
writePipe.WriteString(strings.Join(args, " "))
writePipe.Close()

// 子进程：从 fd[3] 读命令
pipe := os.NewFile(uintptr(3), "pipe")
msg, _ := io.ReadAll(pipe)
cmdArray := strings.Split(string(msg), " ")
syscall.Exec(cmdArray[0], cmdArray, os.Environ())
```

**为什么用管道**：子进程在 clone 时尚未执行用户命令，父进程需要先完成 cgroup 设置，再告知子进程运行什么。管道提供了天然的同步机制。

**验证目标**：`./mydocker run sh` / `./mydocker run top` 均可正常运行。

---

### Step 4：集成 Cgroups 资源限制

目标：支持 `-m 100m` 限制内存。

```go
// 1. 找到 cgroup 挂载点
func findCgroupMount(subsystem string) string {
    // 读 /proc/self/mountinfo，找含 subsystem 的行
}

// 2. 创建 cgroup 目录并设置限制
cgroupPath := filepath.Join(cgroupRoot, "mydocker-cgroup")
os.MkdirAll(cgroupPath, 0755)
os.WriteFile(filepath.Join(cgroupPath, "memory.limit_in_bytes"),
    []byte("104857600"), 0644)  // 100MB

// 3. 将进程加入 cgroup
os.WriteFile(filepath.Join(cgroupPath, "tasks"),
    []byte(strconv.Itoa(pid)), 0644)
```

**验证目标**：

```bash
sudo ./mydocker run -m 10m sh
# 容器内运行内存超限程序，应被 OOM Kill
```

**注意 cgroups v2**：检查 `/sys/fs/cgroup/cgroup.controllers` 是否存在，v2 的配置文件路径和名称不同（如 `memory.max` 而非 `memory.limit_in_bytes`）。

---

### Step 5：构建 Union Filesystem（容器文件系统）

目标：使用 overlay2 实现镜像只读层 + 容器读写层。

```bash
# 准备镜像层（从 Docker 导出）
docker export $(docker create ubuntu) | tar -C /root/ubuntu -xvf -

# 创建工作目录
mkdir -p /root/writeLayer/mycontainer /root/work/mycontainer /root/mnt/mycontainer

# overlay2 挂载
mount -t overlay overlay \
  -o lowerdir=/root/ubuntu,upperdir=/root/writeLayer/mycontainer,workdir=/root/work/mycontainer \
  /root/mnt/mycontainer
```

Go 代码中：

```go
_, err := exec.Command("mount", "-t", "overlay", "overlay",
    "-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work),
    mntPoint).CombinedOutput()
```

**实现 pivot_root 切换容器 rootfs**（详见 `container/init_linux.go:65-95`）：

1. bind mount rootfs 到自身
2. 创建 `.pivot_root` 目录
3. `syscall.PivotRoot(rootfs, pivotDir)`
4. `syscall.Chdir("/")`
5. 卸载并删除 `.pivot_root`

**验证目标**：容器内 `ls /` 显示 ubuntu 文件系统，容器内写文件不影响镜像层。

---

### Step 6：容器元信息管理（ps/stop/rm）

以 JSON 文件持久化容器状态：

```
/var/run/mydocker/
└── <containerName>/
    ├── config.json   # 容器元信息（PID、状态、命令等）
    └── container.log # 容器标准输出（detach 模式）
```

实现：

- `run`：写 config.json，status=running
- `stop`：`kill(pid, SIGTERM)`，更新 status=stopped，清空 pid
- `rm`：检查 status=stopped，删除 config.json + 工作目录
- `ps`：遍历 `/var/run/mydocker/` 读取所有 config.json

---

### Step 7：实现 exec（进入运行中容器）

**关键难点**：需要 cgo 在 Go runtime 启动前调用 `setns()`。

```c
// nsenter/nsenter.go (cgo)
#include <sched.h>
#include <stdio.h>

__attribute__((constructor)) void nsenter_init() {
    char *pid = getenv("mydocker_pid");
    char *cmd = getenv("mydocker_cmd");
    if (pid == NULL || cmd == NULL) return;

    char nspath[1024];
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};
    for (int i = 0; i < 5; i++) {
        snprintf(nspath, sizeof(nspath), "/proc/%s/ns/%s", pid, namespaces[i]);
        int fd = open(nspath, O_RDONLY);
        setns(fd, 0);
        close(fd);
    }
    execl("/bin/sh", "sh", "-c", cmd, NULL);
}
```

父进程通过环境变量传递 pid 和 cmd，然后 `/proc/self/exe exec` 重新执行自身，cgo constructor 在 Go main 前检测到环境变量并进入 namespace 执行命令。

---

### Step 8：实现容器网络

```
1. network create
   └─ 创建 Bridge 设备（netlink）
   └─ 配置 Bridge IP
   └─ 配置 iptables MASQUERADE

2. run --net <network>
   └─ IPAM 分配 IP
   └─ 创建 veth pair
   └─ 一端加入 Bridge
   └─ 另一端移入容器 Network Namespace（netlink.LinkSetNsPid）
   └─ 容器内配置 IP 和默认路由
```

关键 netlink 操作：

```go
// 创建 veth pair
veth := &netlink.Veth{LinkAttrs: la, PeerName: "cif-" + id}
netlink.LinkAdd(veth)

// 将 veth 一端移入容器 namespace
peer, _ := netlink.LinkByName("cif-" + id)
netlink.LinkSetNsPid(peer, containerPid)
```

---

### Step 9：实现数据卷挂载

```go
// -v /host/path:/container/path
func MountVolume(hostPath, containerPath, containerName string) {
    os.MkdirAll(hostPath, 0777)
    containerFullPath := filepath.Join(mntURL, containerPath)
    os.MkdirAll(containerFullPath, 0777)

    // bind mount
    exec.Command("mount", "--bind", hostPath, containerFullPath).Run()
}
```

---

### Step 10：实现 commit（容器打包为镜像）

```go
// 将容器读写层打包为 tar
func CommitContainer(containerName, imageName string) {
    mntURL := fmt.Sprintf("/root/mnt/%s", containerName)
    imageTar := fmt.Sprintf("/root/%s.tar", imageName)
    exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").Run()
}
```

---

## 五、学习路线图

```
Week 1: Namespace + 管道通信
    → 实现 Step 1-3，能运行 ./mydocker run sh

Week 2: Cgroups + 文件系统
    → 实现 Step 4-5，有资源限制和独立文件系统

Week 3: 容器生命周期管理
    → 实现 Step 6，支持 detach 模式和 ps/stop/rm

Week 4: exec + 网络
    → 实现 Step 7-8，能进入容器和容器间通信

Week 5: 数据卷 + commit
    → 实现 Step 9-10，完整功能

Week 6: 深入研究
    → 阅读 runc 源码（OCI 标准实现）
    → 研究 containerd 架构
    → 了解 Docker daemon 和 shim 机制
```

---

## 六、关键技术对比：本项目 vs 真实 Docker

| 特性         | 本项目              | 真实 Docker                       |
| ------------ | ------------------- | --------------------------------- |
| 文件系统     | AUFS                | overlay2（默认）                  |
| Cgroups      | v1                  | v1/v2 自适应                      |
| 容器 runtime | 直接 clone()        | OCI runtime（runc）               |
| 镜像格式     | tar 包              | OCI Image Spec（分层 manifest）   |
| 网络         | 简单 Bridge         | CNM 模型，多种驱动                |
| 安全         | 无 seccomp/AppArmor | seccomp + AppArmor + capabilities |
| 守护进程     | 无                  | dockerd + containerd + shim       |

---

## 七、参考资料

- **《自己动手写 Docker》** — 本仓库的原始参考书（xianlubird 著）
- Linux man pages：`clone(2)`, `unshare(1)`, `pivot_root(2)`, `setns(2)`
- [Linux Kernel Cgroups 文档](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html)
- [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec) — runc 遵循的标准
- [runc 源码](https://github.com/opencontainers/runc) — 生产级容器 runtime 参考
- [《Linux Containers and Virtualization》](https://learning.oreilly.com/library/view/linux-containers-and/9781484262832/) — 深入原理
