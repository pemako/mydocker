package fs2

import (
	"fmt"
	"os"
	"path"

	"github.com/pemako/mydocker/cgroups/subsystems"
)

// MemorySubSystem cgroup v2 内存限制
type MemorySubSystem struct{}

func (s *MemorySubSystem) Name() string { return "memory" }

func (s *MemorySubSystem) Set(cgroupPath string, res *subsystems.ResourceConfig) error {
	if res.MemoryLimit == "" {
		return nil
	}
	subCgroupPath, err := getCgroupPath(cgroupPath, true)
	if err != nil {
		return err
	}
	// v2 使用 memory.max 替代 v1 的 memory.limit_in_bytes
	if err := os.WriteFile(path.Join(subCgroupPath, "memory.max"), []byte(res.MemoryLimit), 0644); err != nil {
		return fmt.Errorf("set cgroup memory fail %v", err)
	}
	return nil
}

func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	return applyCgroup(pid, cgroupPath)
}

func (s *MemorySubSystem) Remove(cgroupPath string) error {
	subCgroupPath, err := getCgroupPath(cgroupPath, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(subCgroupPath)
}
