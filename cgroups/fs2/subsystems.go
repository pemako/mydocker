package fs2

import "github.com/pemako/mydocker/cgroups/subsystems"

// Subsystems cgroup v2 子系统实例列表
var Subsystems = []subsystems.Subsystem{
	&CpusetSubSystem{},
	&MemorySubSystem{},
	&CpuSubSystem{},
}
