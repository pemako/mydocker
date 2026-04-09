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
type BridgeNetworkDriver struct{}

func (d *BridgeNetworkDriver) Name() string { return "bridge" }

// Create 创建 Bridge 网络
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, cidr, _ := net.ParseCIDR(subnet)
	cidr.IP = ip
	n := &Network{
		Name:    name,
		IPRange: cidr,
		Driver:  d.Name(),
	}
	if err := d.initBridge(n); err != nil {
		return nil, fmt.Errorf("failed to create bridge network: %v", err)
	}
	return n, nil
}

// Delete 删除网络：清理路由、iptables 规则，删除网桥
func (d *BridgeNetworkDriver) Delete(network *Network) error {
	if err := deleteIPRoute(network.Name, network.IPRange.String()); err != nil {
		log.Errorf("clean route rule failed after bridge [%s] deleted: %v", network.Name, err)
	}
	if err := configIPTables(network.Name, network.IPRange, true); err != nil {
		log.Errorf("clean snat iptables rule failed after bridge [%s] deleted: %v", network.Name, err)
	}
	br, err := netlink.LinkByName(network.Name)
	if err != nil {
		return fmt.Errorf("get bridge %s failed: %v", network.Name, err)
	}
	return netlink.LinkDel(br)
}

// Connect 将容器端点连接到网桥（创建 veth pair，一端挂到网桥）
func (d *BridgeNetworkDriver) Connect(networkName string, endpoint *Endpoint) error {
	br, err := netlink.LinkByName(networkName)
	if err != nil {
		return err
	}

	la := netlink.NewLinkAttrs()
	la.Name = endpoint.ID[:5]
	la.MasterIndex = br.Attrs().Index

	// 设置 endpoint.Device，以便调用方可通过 PeerName 配置容器内网卡
	veth := netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}
	endpoint.Device = veth
	if err = netlink.LinkAdd(&veth); err != nil {
		return fmt.Errorf("error add endpoint device: %v", err)
	}
	if err = netlink.LinkSetUp(&veth); err != nil {
		return fmt.Errorf("error set endpoint device up: %v", err)
	}
	return nil
}

// Disconnect 从网桥解绑并删除 veth pair
func (d *BridgeNetworkDriver) Disconnect(endpointID string) error {
	vethName := endpointID[:5]
	veth, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("find veth [%s] failed: %v", vethName, err)
	}
	if err = netlink.LinkSetNoMaster(veth); err != nil {
		return fmt.Errorf("unmaster veth [%s] failed: %v", vethName, err)
	}
	if err = netlink.LinkDel(veth); err != nil {
		return fmt.Errorf("delete veth [%s] failed: %v", vethName, err)
	}

	veth2Name := "cif-" + vethName
	veth2, err := netlink.LinkByName(veth2Name)
	if err != nil {
		return fmt.Errorf("find veth [%s] failed: %v", veth2Name, err)
	}
	if err = netlink.LinkDel(veth2); err != nil {
		return fmt.Errorf("delete veth [%s] failed: %v", veth2Name, err)
	}
	return nil
}

// initBridge 初始化 Linux Bridge：创建设备、设置 IP、启动、配置 iptables SNAT
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	if err := createBridgeInterface(n.Name); err != nil {
		return fmt.Errorf("create bridge %s error: %v", n.Name, err)
	}
	gatewayIP := *n.IPRange
	gatewayIP.IP = n.IPRange.IP
	if err := setInterfaceIP(n.Name, gatewayIP.String()); err != nil {
		return fmt.Errorf("set bridge ip %s on bridge %s error: %v", gatewayIP.String(), n.Name, err)
	}
	if err := setInterfaceUP(n.Name); err != nil {
		return fmt.Errorf("set bridge %s up error: %v", n.Name, err)
	}
	if err := configIPTables(n.Name, n.IPRange, false); err != nil {
		return fmt.Errorf("set iptables for %s error: %v", n.Name, err)
	}
	return nil
}

// createBridgeInterface 创建 Bridge 虚拟设备
func createBridgeInterface(bridgeName string) error {
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}
	if err = netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("create bridge %s error: %v", bridgeName, err)
	}
	return nil
}

// setInterfaceIP 设置网络接口的 IP 地址
func setInterfaceIP(name string, rawIP string) error {
	var (
		iface netlink.Link
		err   error
	)
	for i := 0; i < 2; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("get link %s failed: %v", name, err)
	}
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

// deleteIPRoute 删除接口上对应子网的路由
func deleteIPRoute(name string, rawIP string) error {
	var (
		iface netlink.Link
		err   error
	)
	for i := 0; i < 2; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("get link %s failed: %v", name, err)
	}
	list, err := netlink.RouteList(iface, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	for _, route := range list {
		if route.Dst != nil && route.Dst.String() == rawIP {
			if err = netlink.RouteDel(&route); err != nil {
				log.Errorf("route [%v] del failed: %v", route, err)
			}
		}
	}
	return nil
}

// setInterfaceUP 启动网络接口
func setInterfaceUP(interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return fmt.Errorf("get link %s failed: %v", interfaceName, err)
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set link %s up failed: %v", interfaceName, err)
	}
	return nil
}

// configIPTables 配置/删除 SNAT iptables 规则
// iptables -t nat [-A|-D] POSTROUTING -s {subnet} ! -o {bridge} -j MASQUERADE
func configIPTables(bridgeName string, subnet *net.IPNet, isDelete bool) error {
	action := "-A"
	if isDelete {
		action = "-D"
	}
	iptablesCmd := fmt.Sprintf("-t nat %s POSTROUTING -s %s ! -o %s -j MASQUERADE",
		action, subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	log.Infof("configIPTables cmd: %v", cmd.String())
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables output: %v", string(output))
	}
	return err
}
