// Package nsenter 通过 CGO 实现容器 Namespace 的进入（setns 系统调用）。
//
// # 工作原理
//
// Go 运行时在启动时会创建多个线程，而 setns(2) 只能作用于调用线程本身。
// 为了在 Go 程序启动的最早阶段（线程数最少时）完成 namespace 切换，
// 本包利用 CGO 的 __attribute__((constructor)) 机制：
// C 函数 enter_namespace 在 Go runtime 初始化之前由动态链接器自动调用。
//
// # 环境变量协议
//
// exec 命令通过以下环境变量向子进程传递目标容器信息：
//
//   - mydocker_pid — 目标容器的 init 进程 PID（对应 /proc/<pid>/ns/）
//   - mydocker_cmd — 要在容器内执行的命令字符串
//
// 当上述环境变量存在时，enter_namespace 会依次调用 setns 进入目标容器的
// MNT、NET、UTS、IPC、PID namespace，然后 exec 用户命令。
// 当环境变量不存在时（正常 mydocker 启动流程），该函数立即返回，不产生任何影响。
//
// # 构建约束
//
// 本包仅在 linux && cgo 环境下编译真实实现；
// 其他平台使用 nsenter_stub.go 中的空实现。
package nsenter
