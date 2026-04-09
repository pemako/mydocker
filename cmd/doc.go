// Package cmd 实现 mydocker 的所有 CLI 子命令。
//
// 本包基于 [github.com/spf13/cobra] 构建命令树，每个文件对应一个子命令：
//
//   - run.go     — 创建并运行容器，支持资源限制、数据卷、网络、环境变量等参数
//   - stop.go    — 向容器发送 SIGTERM 信号并更新容器状态
//   - rm.go      — 删除已停止的容器；-f 标志可强制删除运行中的容器并清理网络
//   - ps.go      — 列出所有容器及其状态
//   - exec.go    — 通过 nsenter 进入运行中容器的 Namespace 执行命令
//   - logs.go    — 读取并输出后台容器的日志文件
//   - commit.go  — 将容器的 merged 层打包为新镜像 tar 包
//   - inspect.go — 将容器元数据以格式化 JSON 输出
//   - restart.go — 停止容器后使用保存的配置重新启动
//   - network.go — network 子命令组（create / list / remove）
//   - init.go    — init 内部命令，由容器进程自身调用，不对外暴露
//   - root.go    — 根命令及子命令注册
package cmd
