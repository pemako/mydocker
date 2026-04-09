package cgroups

import (
	"github.com/pemako/mydocker/cgroups/subsystems"
	"github.com/sirupsen/logrus"
)

// CgroupManager cgroup 管理器接口
type CgroupManager interface {
	Apply(pid int) error
	Set(res *subsystems.ResourceConfig) error
	Destroy() error
}

// NewCgroupManager 根据当前系统自动选择 v1 或 v2 实现
func NewCgroupManager(path string) CgroupManager {
	if IsCgroup2UnifiedMode() {
		logrus.Infof("use cgroup v2")
		return NewCgroupManagerV2(path)
	}
	logrus.Infof("use cgroup v1")
	return NewCgroupManagerV1(path)
}
