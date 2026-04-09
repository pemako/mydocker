//go:build linux

package cgroups

import (
	"sync"
	"syscall"
)

const cgroup2SuperMagic = 0x63677270

var (
	isUnifiedOnce sync.Once
	isUnified     bool
)

// IsCgroup2UnifiedMode 检测当前系统是否使用 cgroup v2
func IsCgroup2UnifiedMode() bool {
	isUnifiedOnce.Do(func() {
		var st syscall.Statfs_t
		err := syscall.Statfs("/sys/fs/cgroup", &st)
		if err != nil {
			isUnified = false
			return
		}
		isUnified = st.Type == cgroup2SuperMagic
	})
	return isUnified
}
