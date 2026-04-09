//go:build !linux

package network

import (
	"fmt"
	"net"

	"github.com/pemako/mydocker/container"
)

func connectImpl(networkName string, info *container.ContainerInfo) (net.IP, error) {
	return nil, fmt.Errorf("container networking is only supported on Linux")
}

func disconnectImpl(networkName string, info *container.ContainerInfo) error {
	return fmt.Errorf("container networking is only supported on Linux")
}
