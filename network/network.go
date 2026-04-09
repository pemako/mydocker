package network

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"text/tabwriter"

	"github.com/pemako/mydocker/container"
	log "github.com/sirupsen/logrus"
)

const defaultNetworkPath = "/var/lib/mydocker/network/network/"

// Network 网络定义
type Network struct {
	Name    string     `json:"name"`    // 网络名称
	IPRange *net.IPNet `json:"ipRange"` // 网络地址段
	Driver  string     `json:"driver"`  // 网络驱动类型
}

// Endpoint 网络端点（容器在网络中的虚拟网卡）
type Endpoint struct {
	ID          string           `json:"id"`
	Device      any              `json:"dev"` // netlink.Veth on Linux
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

// NetworkDriver 网络驱动接口
type NetworkDriver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(network *Network) error
	Connect(networkName string, endpoint *Endpoint) error
	Disconnect(endpointID string) error
}

var drivers = map[string]NetworkDriver{}

func Init() {
	bridgeDriver := BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("check %s failed: %v", defaultNetworkPath, err)
			return
		}
		if err = os.MkdirAll(defaultNetworkPath, 0644); err != nil {
			log.Errorf("create %s failed: %v", defaultNetworkPath, err)
		}
	}
}

func init() { Init() }

// dump 将网络配置持久化到文件
func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(dumpPath, 0644); err != nil {
			return fmt.Errorf("create network dump path %s failed: %v", dumpPath, err)
		}
	}
	netPath := path.Join(dumpPath, nw.Name)
	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open file %s failed: %v", netPath, err)
	}
	defer netFile.Close()

	netJSON, err := json.Marshal(nw)
	if err != nil {
		return fmt.Errorf("marshal network %v failed: %v", nw, err)
	}
	_, err = netFile.Write(netJSON)
	return err
}

// load 从文件加载网络配置
func (nw *Network) load(dumpPath string) error {
	netConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer netConfigFile.Close()

	netJSON := make([]byte, 4096)
	n, err := netConfigFile.Read(netJSON)
	if err != nil {
		return err
	}
	return json.Unmarshal(netJSON[:n], nw)
}

// remove 删除网络配置文件
func (nw *Network) remove(dumpPath string) error {
	fullPath := path.Join(dumpPath, nw.Name)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Remove(fullPath)
}

// loadNetwork 从磁盘加载所有网络配置
func loadNetwork() (map[string]*Network, error) {
	networks := map[string]*Network{}
	err := filepath.Walk(defaultNetworkPath, func(netPath string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		_, netName := path.Split(netPath)
		n := &Network{Name: netName}
		if err = n.load(netPath); err != nil {
			log.Errorf("load network %s error: %v", netName, err)
		}
		networks[netName] = n
		return nil
	})
	return networks, err
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	gatewayIP, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIP

	n, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	return n.dump(defaultNetworkPath)
}

// ListNetwork 列出所有网络
func ListNetwork() {
	networks, err := loadNetwork()
	if err != nil {
		log.Errorf("load network from file failed: %v", err)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIPRange\tDriver\n")
	for _, n := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", n.Name, n.IPRange.String(), n.Driver)
	}
	if err = w.Flush(); err != nil {
		log.Errorf("flush error %v", err)
	}
}

// DeleteNetwork 删除网络
func DeleteNetwork(networkName string) error {
	networks, err := loadNetwork()
	if err != nil {
		return fmt.Errorf("load network from file failed: %v", err)
	}
	n, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network: %s", networkName)
	}
	if err = ipAllocator.Release(n.IPRange, &n.IPRange.IP); err != nil {
		return fmt.Errorf("release network gateway ip failed: %v", err)
	}
	if err = drivers[n.Driver].Delete(n); err != nil {
		return fmt.Errorf("remove network driver error: %v", err)
	}
	return n.remove(defaultNetworkPath)
}

// Connect 连接容器到网络，返回分配的 IP（Linux-specific 实现在 connect_linux.go）
func Connect(networkName string, info *container.ContainerInfo) (net.IP, error) {
	return connectImpl(networkName, info)
}

// Disconnect 将容器从网络中移除（Linux-specific 实现在 connect_linux.go）
func Disconnect(networkName string, info *container.ContainerInfo) error {
	return disconnectImpl(networkName, info)
}
