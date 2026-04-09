// mydocker 是一个用于学习目的的简易容器运行时实现。
//
// 通过本项目可以了解 Docker 的核心工作原理，包括：
//   - Linux Namespace 隔离（UTS、PID、Mount、Network、IPC）
//   - Cgroup v1/v2 资源限制（内存、CPU、CPU 核心绑定）
//   - OverlayFS 联合文件系统
//   - 容器网络（Linux Bridge、veth pair、iptables）
//   - IPAM IP 地址管理
//
// 用法：
//
//	mydocker [command] [flags]
//
// 主要命令：
//
//	run      创建并启动一个新容器
//	stop     停止运行中的容器
//	rm       删除容器（-f 可强制删除运行中的容器）
//	ps       列出所有容器
//	exec     在运行中的容器内执行命令
//	logs     查看容器日志
//	commit   将容器提交为镜像
//	inspect  以 JSON 格式查看容器详细信息
//	restart  重启容器
//	network  管理容器网络（create / list / remove）
package main
