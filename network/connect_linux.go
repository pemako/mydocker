//go:build linux

package network

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pemako/mydocker/container"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// linuxEndpoint extends Endpoint for Linux with the actual netlink.Veth device
type linuxEndpoint struct {
	Endpoint
	VethDevice netlink.Veth
}

func connectImpl(networkName string, info *container.ContainerInfo) (net.IP, error) {
	networks, err := loadNetwork()
	if err != nil {
		return nil, fmt.Errorf("load network from file failed: %v", err)
	}
	n, ok := networks[networkName]
	if !ok {
		return nil, fmt.Errorf("no such network: %s", networkName)
	}

	ip, err := ipAllocator.Allocate(n.IPRange)
	if err != nil {
		return nil, fmt.Errorf("allocate ip error: %v", err)
	}

	ep := &linuxEndpoint{
		Endpoint: Endpoint{
			ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
			IPAddress:   ip,
			Network:     n,
			PortMapping: info.PortMapping,
		},
	}

	if err = drivers[n.Driver].Connect(networkName, &ep.Endpoint); err != nil {
		return ip, err
	}
	// 将 Device 字段转换回 netlink.Veth（bridge_linux.go 中设置的）
	if veth, ok := ep.Endpoint.Device.(netlink.Veth); ok {
		ep.VethDevice = veth
	}

	if err = configEndpointIpAddressAndRoute(ep, info); err != nil {
		return ip, err
	}
	return ip, addPortMapping(ep)
}

func disconnectImpl(networkName string, info *container.ContainerInfo) error {
	networks, err := loadNetwork()
	if err != nil {
		return fmt.Errorf("load network from file failed: %v", err)
	}
	n, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}

	endpointID := fmt.Sprintf("%s-%s", info.Id, networkName)
	drivers[n.Driver].Disconnect(endpointID)

	ep := &linuxEndpoint{
		Endpoint: Endpoint{
			ID:          endpointID,
			IPAddress:   net.ParseIP(info.IP),
			Network:     n,
			PortMapping: info.PortMapping,
		},
	}
	return deletePortMapping(ep)
}

// enterContainerNetNS 进入容器的网络 namespace，返回恢复函数
func enterContainerNetNS(enLink *netlink.Link, info *container.ContainerInfo) func() {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace: %v", err)
	}
	nsFD := f.Fd()
	runtime.LockOSThread()

	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns: %v", err)
	}
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns: %v", err)
	}
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns: %v", err)
	}
	return func() {
		netns.Set(origns)
		origns.Close()
		runtime.UnlockOSThread()
		f.Close()
	}
}

// configEndpointIpAddressAndRoute 在容器网络 namespace 中配置 IP 和路由
func configEndpointIpAddressAndRoute(ep *linuxEndpoint, info *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.VethDevice.PeerName)
	if err != nil {
		return fmt.Errorf("found veth [%s] failed: %v", ep.VethDevice.PeerName, err)
	}

	defer enterContainerNetNS(&peerLink, info)()

	// 配置容器内 veth 端点的 IP（使用网络的 CIDR，IP 为容器分配的 IP）
	interfaceIP := *ep.Network.IPRange
	interfaceIP.IP = ep.IPAddress
	if err = setInterfaceIP(ep.VethDevice.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("set interface ip error: %v", err)
	}
	if err = setInterfaceUP(ep.VethDevice.PeerName); err != nil {
		return err
	}
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}

	// 设置默认路由，所有流量通过网关（bridge IP）转发
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IPRange.IP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("add default route error: %v", err)
	}
	return nil
}

// addPortMapping 添加 iptables DNAT 端口映射规则
func addPortMapping(ep *linuxEndpoint) error {
	return configPortMapping(ep, false)
}

// deletePortMapping 删除 iptables DNAT 端口映射规则
func deletePortMapping(ep *linuxEndpoint) error {
	return configPortMapping(ep, true)
}

// configPortMapping 配置端口映射 iptables 规则
// iptables -t nat [-A|-D] PREROUTING ! -i {bridge} -p tcp -m tcp --dport {hostPort} -j DNAT --to-destination {containerIP}:{containerPort}
func configPortMapping(ep *linuxEndpoint, isDelete bool) error {
	action := "-A"
	if isDelete {
		action = "-D"
	}
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error: %v", pm)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat %s PREROUTING ! -i %s -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			action, ep.Network.Name, portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		log.Infof("port mapping cmd: %v", cmd.String())
		if output, err := cmd.Output(); err != nil {
			log.Errorf("iptables output: %v, error: %v", string(output), err)
		}
	}
	return nil
}
