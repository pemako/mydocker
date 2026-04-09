package fs2

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

// getCgroupPath 返回 cgroup v2 中的绝对路径，autoCreate 为 true 时自动创建目录
func getCgroupPath(cgroupPath string, autoCreate bool) (string, error) {
	absPath := path.Join(UnifiedMountpoint, cgroupPath)
	if !autoCreate {
		return absPath, nil
	}
	_, err := os.Stat(absPath)
	if err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(absPath, 0755)
		return absPath, err
	}
	return absPath, err
}

// applyCgroup 将进程 pid 写入 cgroup.procs
func applyCgroup(pid int, cgroupPath string) error {
	subCgroupPath, err := getCgroupPath(cgroupPath, true)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
	if err = os.WriteFile(path.Join(subCgroupPath, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}
