//go:build !linux

package network

import "fmt"

// BridgeNetworkDriver stub for non-Linux platforms
type BridgeNetworkDriver struct {
}

// Name returns the driver name
func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// Create stub
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	return nil, fmt.Errorf("bridge network driver is only supported on Linux")
}

// Delete stub
func (d *BridgeNetworkDriver) Delete(network Network) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}

// Connect stub
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}

// Disconnect stub
func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return fmt.Errorf("bridge network driver is only supported on Linux")
}
