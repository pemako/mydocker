//go:build !linux

package container

import (
	"fmt"
	"os"
	"os/exec"
)

// NewParentProcess stub for non-Linux platforms
func NewParentProcess(tty bool, containerName, volume, imageName string, envSlice []string) (*exec.Cmd, *os.File) {
	fmt.Println("Container functionality is only supported on Linux")
	return nil, nil
}
