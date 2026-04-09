// Package cgroups 提供对 Linux cgroup v1 和 v2 的统一资源限制管理。
//
// # 版本自动检测
//
// [NewCgroupManager] 是唯一的公开工厂函数。它通过 [IsCgroup2UnifiedMode] 检测
// /sys/fs/cgroup 的文件系统类型（magic number 0x63677270），自动选择对应实现：
//   - cgroup v1 → [NewCgroupManagerV1]，使用 [subsystems] 包中的子系统
//   - cgroup v2 → [NewCgroupManagerV2]，使用 [fs2] 包中的子系统
//
// # CgroupManager 接口
//
//	type CgroupManager interface {
//	    Apply(pid int) error                       // 将进程加入 cgroup
//	    Set(res *subsystems.ResourceConfig) error  // 设置资源限制
//	    Destroy() error                            // 删除 cgroup
//	}
//
// # 子包
//
//   - [github.com/pemako/mydocker/cgroups/subsystems] — cgroup v1 子系统实现
//   - [github.com/pemako/mydocker/cgroups/fs2]        — cgroup v2 子系统实现
//
// # 典型用法
//
//	mgr := cgroups.NewCgroupManager("mydocker-cgroup")
//	defer mgr.Destroy()
//	mgr.Set(&subsystems.ResourceConfig{MemoryLimit: "100m"})
//	mgr.Apply(pid)
package cgroups
