// Package container 负责容器的生命周期管理与文件系统构建。
//
// # 容器元数据
//
// [ContainerInfo] 保存容器的完整运行时信息（PID、状态、镜像、环境变量、网络等），
// 持久化到 /var/run/mydocker/<name>/config.json。
// 主要操作函数：
//   - [RecordContainerInfo]   — 创建并写入容器元数据
//   - [GetContainerInfoByName] — 按名称读取容器信息
//   - [UpdateContainerInfo]   — 将修改后的元数据写回磁盘
//   - [DeleteContainerInfo]   — 删除容器信息目录
//   - [ListContainers]        — 列出所有容器
//
// # 容器状态
//
// 容器有三种状态常量：[RUNNING]、[STOP]、[EXIT]。
//
// # 文件系统（OverlayFS）
//
// 容器根文件系统基于 OverlayFS 四层结构构建：
//
//	/var/lib/mydocker/overlay2/<id>/
//	  lower/   只读层，由镜像 tar 包解压而来
//	  upper/   读写层，容器内的所有写操作落在此处
//	  work/    OverlayFS 所需的工作目录
//	  merged/  联合挂载点，作为容器的根目录
//
// 镜像 tar 包存放于 /var/lib/mydocker/image/<name>.tar。
// 主要函数：[NewWorkSpace]、[DeleteWorkSpace]。
//
// # 容器进程
//
// [NewParentProcess] 使用 /proc/self/exe init 自举方式创建隔离进程，
// 通过匿名管道向子进程传递初始命令，并通过 Namespace clone flags 实现隔离。
// [RunContainerInitProcess] 在子进程侧执行：完成 mount namespace 初始化后
// 通过 syscall.Exec 替换自身为用户指定的进程。
package container
