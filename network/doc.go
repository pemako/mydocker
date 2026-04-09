// Package network 实现容器的网络管理功能。
//
// # 核心概念
//
//   - [Network]        — 一个虚拟网络，由 Linux Bridge 设备承载，关联一个子网 CIDR
//   - [Endpoint]       — 容器在网络中的连接点，对应一对 veth 设备
//   - [NetworkDriver]  — 网络驱动接口，目前仅实现 bridge 驱动（[BridgeNetworkDriver]）
//   - [IPAM]           — IP 地址管理器，使用位图算法在子网内分配/释放 IP
//
// # 网络持久化
//
// 网络配置持久化到 /var/lib/mydocker/network/network/<name>（JSON 格式），
// 重启后通过 loadNetwork 自动恢复。IPAM 分配状态持久化到
// /var/lib/mydocker/network/ipam/subnet.json。
//
// # 容器连接流程（Linux）
//
//  1. [Connect] 从网络子网分配 IP，创建 [Endpoint]
//  2. [BridgeNetworkDriver.Connect] 创建 veth pair，主机端挂到 Bridge
//  3. configEndpointIpAddressAndRoute 进入容器 net namespace，
//     为容器端 veth 配置 IP 和默认路由
//  4. addPortMapping 通过 iptables DNAT 规则配置端口映射
//
// # Bridge 驱动
//
// [BridgeNetworkDriver] 使用 [github.com/vishvananda/netlink] 操作内核网络接口：
//   - 创建/删除 Linux Bridge 设备
//   - 创建/删除 veth pair 并挂载到 Bridge
//   - 配置 iptables MASQUERADE（SNAT）规则使容器访问外网
//
// # 典型用法
//
//	// 创建网络
//	network.CreateNetwork("bridge", "192.168.0.0/24", "mynet")
//
//	// 连接容器
//	ip, err := network.Connect("mynet", containerInfo)
//
//	// 断开并清理
//	network.Disconnect("mynet", containerInfo)
package network
