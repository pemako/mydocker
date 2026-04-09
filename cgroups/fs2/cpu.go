package fs2

import (
	"fmt"
	"os"
	"path"

	"github.com/pemako/mydocker/cgroups/subsystems"
	log "github.com/sirupsen/logrus"
)

// CpuSubSystem cgroup v2 CPU 限制
// 注意: v2 不支持 cpu.shares，CpuShare 字段在 v2 中被忽略
type CpuSubSystem struct{}

func (s *CpuSubSystem) Name() string { return "cpu" }

const (
	periodDefault = 100000
	percent       = 100
)

func (s *CpuSubSystem) Set(cgroupPath string, res *subsystems.ResourceConfig) error {
	if res.CpuShare != "" {
		log.Warnf("cgroup v2 does not support cpu.shares, --cpushare flag is ignored")
	}
	// v2 暂不设置 CPU 配额（需要 CpuCfsQuota 字段，当前 ResourceConfig 仅有 CpuShare）
	return nil
}

func (s *CpuSubSystem) Apply(cgroupPath string, pid int) error {
	return applyCgroup(pid, cgroupPath)
}

func (s *CpuSubSystem) Remove(cgroupPath string) error {
	subCgroupPath, err := getCgroupPath(cgroupPath, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(subCgroupPath)
}

// setCpuMax 设置 cpu.max，quota 单位为百分比（如 20 表示 20%）
func setCpuMax(cgroupPath string, quotaPercent int) error {
	subCgroupPath, err := getCgroupPath(cgroupPath, true)
	if err != nil {
		return err
	}
	quota := periodDefault / percent * quotaPercent
	value := fmt.Sprintf("%d %d", quota, periodDefault)
	if err = os.WriteFile(path.Join(subCgroupPath, "cpu.max"), []byte(value), 0644); err != nil {
		return fmt.Errorf("set cgroup cpu.max fail %v", err)
	}
	return nil
}
