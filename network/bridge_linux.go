//go:build linux

package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// BridgeNetworkDriver Linux Bridge 网络驱动
type BridgeNetworkDriver struct {
}

// Name 驱动名
func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// Create 创建 Bridge 网络
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, cidr, _ := net.ParseCIDR(subnet)
	cidr.IP = ip

	n := &Network{
		Name:    name,
		IpRange: cidr,
		Driver:  d.Name(),
	}

	err := d.initBridge(n)
	if err != nil {
		log.Errorf("error init bridge: %v", err)
	}

	return n, err
}

// Delete 删除网络
func (d *BridgeNetworkDriver) Delete(network Network) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// Connect 连接容器到网络
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 创建 Veth 接口的配置
	la := netlink.NewLinkAttrs()
	// Linux 接口名限制在15个字符以内
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	// 创建 Veth 对，一端连接到网桥
	endpoint.Device = la.Name
	veth := &netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}
	if err = netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}

	// 启动 veth
	if err = netlink.LinkSetUp(veth); err != nil {
		return fmt.Errorf("error Set Endpoint Device Up: %v", err)
	}
	return nil
}

// Disconnect 从网络断开容器
func (d *BridgeNetworkDriver) Disconnect(network Network, endpoint *Endpoint) error {
	return nil
}

// initBridge 初始化 Linux Bridge
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	// 创建 Bridge 虚拟设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("error create bridge %s: %v", bridgeName, err)
	}

	// 设置 Bridge 设备的地址和路由
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP

	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("error set bridge ip: %s on bridge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}

	// 启动 Bridge 设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("error set bridge up: %s: %v", bridgeName, err)
	}

	// 设置 iptables 的 SNAT 规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}

// createBridgeInterface 创建 Bridge 设备
func createBridgeInterface(bridgeName string) error {
	// 检查是否已存在
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 创建 Link 对象
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 使用 netlink 库创建 Bridge 对象
	br := &netlink.Bridge{LinkAttrs: la}
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s: %v", bridgeName, err)
	}
	return nil
}

// setInterfaceIP 设置网络接口的 IP 地址
func setInterfaceIP(name string, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot the error: %v", err)
	}

	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	addr := &netlink.Addr{IPNet: ipNet, Peer: ipNet, Label: "", Flags: 0, Scope: 0, Broadcast: nil}
	return netlink.AddrAdd(iface, addr)
}

// setInterfaceUP 启动网络接口
func setInterfaceUP(interfaceName string) error {
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("error retrieving a link named [ %s ]: %v", interfaceName, err)
	}

	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("error enabling interface for %s: %v", interfaceName, err)
	}
	return nil
}

// setupIPTables 设置 iptables 对应规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 创建 MASQUERADE 规则，实现容器访问外网
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}
