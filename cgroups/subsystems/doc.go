// Package subsystems 实现 cgroup v1 的各资源子系统。
//
// 所有子系统均实现 [Subsystem] 接口：
//
//	type Subsystem interface {
//	    Name() string
//	    Set(path string, res *ResourceConfig) error   // 写入资源限制文件
//	    Apply(path string, pid int) error              // 将 PID 写入 tasks
//	    Remove(path string) error                      // 删除 cgroup 目录
//	}
//
// 已实现的子系统：
//
//   - [MemorySubSystem] — 写入 memory.limit_in_bytes 限制内存用量
//   - [CpuSubSystem]    — 写入 cpu.shares 设置 CPU 相对权重
//   - [CpusetSubSystem] — 写入 cpuset.cpus 绑定 CPU 核心
//
// [ResourceConfig] 汇总所有子系统的配置项，由上层统一传入。
//
// cgroup 根挂载点通过解析 /proc/self/mountinfo 动态查找（[FindCgroupMountpoint]），
// 因此无需硬编码路径，兼容不同发行版的挂载位置。
package subsystems
