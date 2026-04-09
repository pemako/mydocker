package cgroups

import (
	"github.com/pemako/mydocker/cgroups/subsystems"
	"github.com/sirupsen/logrus"
)

// CgroupManagerV1 cgroup v1 管理器
type CgroupManagerV1 struct {
	Path string
}

func NewCgroupManagerV1(path string) *CgroupManagerV1 {
	return &CgroupManagerV1{Path: path}
}

// Apply 将进程 pid 加入到 cgroup
func (c *CgroupManagerV1) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Apply(c.Path, pid); err != nil {
			logrus.Errorf("apply subsystem %s err: %v", subSysIns.Name(), err)
		}
	}
	return nil
}

// Set 设置 cgroup 资源限制
func (c *CgroupManagerV1) Set(res *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Set(c.Path, res); err != nil {
			logrus.Errorf("set subsystem %s err: %v", subSysIns.Name(), err)
		}
	}
	return nil
}

// Destroy 释放 cgroup
func (c *CgroupManagerV1) Destroy() error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			logrus.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
