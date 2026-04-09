// Package fs2 实现 cgroup v2（unified hierarchy）的各资源子系统。
//
// cgroup v2 将所有子系统统一挂载在 /sys/fs/cgroup（[UnifiedMountpoint]），
// 不再像 v1 那样按子系统分别挂载，进程通过写入 cgroup.procs（而非 tasks）加入 cgroup。
//
// 已实现的子系统：
//
//   - [MemorySubSystem] — 写入 memory.max 限制内存用量（替代 v1 的 memory.limit_in_bytes）
//   - [CpuSubSystem]    — v2 不支持 cpu.shares，当前仅记录警告日志；
//     如需 CPU 配额限制可通过内部 setCpuMax 写入 cpu.max
//   - [CpusetSubSystem] — 写入 cpuset.cpus 绑定 CPU 核心（与 v1 同名文件）
//
// 所有子系统均实现 [github.com/pemako/mydocker/cgroups/subsystems.Subsystem] 接口，
// 与 cgroup v1 子系统可互换使用。
package fs2
