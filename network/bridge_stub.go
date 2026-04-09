//go:build !linux

package network

import "fmt"

// BridgeNetworkDriver stub for non-Linux platforms
type BridgeNetworkDriver struct{}

func (d *BridgeNetworkDriver) Name() string { return "bridge" }

func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	return nil, fmt.Errorf("bridge network driver is only supported on Linux")
}

func (d *BridgeNetworkDriver) Delete(network *Network) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}

func (d *BridgeNetworkDriver) Connect(networkName string, endpoint *Endpoint) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}

func (d *BridgeNetworkDriver) Disconnect(endpointID string) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}
