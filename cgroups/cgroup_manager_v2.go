package cgroups

import (
	"github.com/pemako/mydocker/cgroups/fs2"
	"github.com/pemako/mydocker/cgroups/subsystems"
	"github.com/sirupsen/logrus"
)

// CgroupManagerV2 cgroup v2 管理器
type CgroupManagerV2 struct {
	Path string
}

func NewCgroupManagerV2(path string) *CgroupManagerV2 {
	return &CgroupManagerV2{Path: path}
}

// Apply 将进程 pid 加入到 cgroup v2
func (c *CgroupManagerV2) Apply(pid int) error {
	for _, subSysIns := range fs2.Subsystems {
		if err := subSysIns.Apply(c.Path, pid); err != nil {
			logrus.Errorf("apply subsystem %s err: %v", subSysIns.Name(), err)
		}
	}
	return nil
}

// Set 设置 cgroup v2 资源限制
func (c *CgroupManagerV2) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range fs2.Subsystems {
		if err := subSysIns.Set(c.Path, res); err != nil {
			logrus.Errorf("set subsystem %s err: %v", subSysIns.Name(), err)
		}
	}
	return nil
}

// Destroy 释放 cgroup v2
func (c *CgroupManagerV2) Destroy() error {
	for _, subSysIns := range fs2.Subsystems {
		if err := subSysIns.Remove(c.Path); err != nil {
			logrus.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
