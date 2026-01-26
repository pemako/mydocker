package container

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// GetPidFromPidStr 将 PID 字符串转换为整数
func GetPidFromPidStr(pidStr string) (int, error) {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("convert pid from string to int error: %v", err)
	}
	return pid, nil
}

// KillProcess 向进程发送 SIGTERM 信号
func KillProcess(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("kill process %d error: %v", pid, err)
	}
	return nil
}

// PathExists 检查路径是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
