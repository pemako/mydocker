//go:build !linux

package container

import "fmt"

// RunContainerInitProcess stub for non-Linux platforms
func RunContainerInitProcess() error {
	return fmt.Errorf("container init process is only supported on Linux")
}
