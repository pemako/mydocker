package network

import (
	"net"
)

// Network 网络定义
type Network struct {
	Name    string     // 网络名称
	IpRange *net.IPNet // 网络地址段
	Driver  string     // 网络驱动类型
}

// Endpoint 网络端点（容器在网络中的虚拟网卡）
type Endpoint struct {
	ID          string           // 端点ID
	Device      string           // veth设备名
	IPAddress   net.IP           // IP地址
	MacAddress  net.HardwareAddr // MAC地址
	Network     *Network         // 所属网络
	PortMapping []string         // 端口映射
}

// NetworkDriver 网络驱动接口
type NetworkDriver interface {
	// Name 驱动名称
	Name() string
	// Create 创建网络
	Create(subnet string, name string) (*Network, error)
	// Delete 删除网络
	Delete(network Network) error
	// Connect 连接容器到网络
	Connect(network *Network, endpoint *Endpoint) error
	// Disconnect 断开容器网络连接
	Disconnect(network Network, endpoint *Endpoint) error
}

var (
	// drivers 网络驱动集合
	drivers = map[string]NetworkDriver{}
	// networks 网络集合
	networks = map[string]*Network{}
)

// Init 初始化网络
func Init() error {
	// 加载网络驱动
	bridgeDriver := BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 加载已有网络配置
	// TODO: 从文件加载已创建的网络
	return nil
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 解析子网地址
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}

	// 分配网关IP（默认使用第一个IP）
	gatewayIP, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIP

	// 调用网络驱动创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	// 保存网络信息
	networks[name] = nw
	return nil
}

// Connect 连接容器到网络
func Connect(networkName string, containerID string) error {
	// 获取网络
	network, ok := networks[networkName]
	if !ok {
		return nil
	}

	// 分配IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// 创建网络端点
	ep := &Endpoint{
		ID:        containerID,
		IPAddress: ip,
		Network:   network,
	}

	// 调用网络驱动连接
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	return nil
}

// Disconnect 断开容器网络
func Disconnect(networkName string, endpoint *Endpoint) error {
	network, ok := networks[networkName]
	if !ok {
		return nil
	}
	return drivers[network.Driver].Disconnect(*network, endpoint)
}

// ListNetwork 列出所有网络
func ListNetwork() {
	// TODO: 实现网络列表功能
	// 目前仅打印内存中的网络
	for name, nw := range networks {
		println("Network:", name, "Driver:", nw.Driver, "Subnet:", nw.IpRange.String())
	}
}

// DeleteNetwork 删除网络
func DeleteNetwork(networkName string) error {
	network, ok := networks[networkName]
	if !ok {
		return nil
	}

	// 调用驱动删除网络
	if err := drivers[network.Driver].Delete(*network); err != nil {
		return err
	}

	// 从内存中移除
	delete(networks, networkName)
	return nil
}
