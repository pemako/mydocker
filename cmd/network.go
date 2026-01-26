package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/network"
)

// networkCmd 网络管理命令
var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage container networks",
	Long:  "Create, list, and remove container networks",
}

// networkCreateCmd 创建网络命令
var networkCreateCmd = &cobra.Command{
	Use:   "create [flags] NETWORK_NAME",
	Short: "Create a container network",
	Long:  "Create a new container network with the specified driver and subnet",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing network name")
		}

		networkName := args[0]
		driver, _ := cmd.Flags().GetString("driver")
		subnet, _ := cmd.Flags().GetString("subnet")

		if subnet == "" {
			return fmt.Errorf("subnet is required")
		}

		err := network.CreateNetwork(driver, subnet, networkName)
		if err != nil {
			return fmt.Errorf("create network error: %v", err)
		}
		log.Infof("Network %s created successfully", networkName)
		return nil
	},
}

// networkListCmd 列出所有网络
var networkListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all container networks",
	Long:  "List all container networks",
	RunE: func(cmd *cobra.Command, args []string) error {
		network.Init()
		network.ListNetwork()
		return nil
	},
}

// networkRemoveCmd 删除网络
var networkRemoveCmd = &cobra.Command{
	Use:   "remove NETWORK_NAME",
	Short: "Remove a container network",
	Long:  "Remove a container network",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing network name")
		}
		network.Init()
		err := network.DeleteNetwork(args[0])
		if err != nil {
			return fmt.Errorf("remove network error: %v", err)
		}
		return nil
	},
}

func init() {
	// network create 命令的标志
	networkCreateCmd.Flags().StringP("driver", "d", "bridge", "Network driver")
	networkCreateCmd.Flags().StringP("subnet", "s", "", "Subnet CIDR (e.g., 192.168.0.0/24)")

	// 添加 network 子命令
	networkCmd.AddCommand(networkCreateCmd)
	networkCmd.AddCommand(networkListCmd)
	networkCmd.AddCommand(networkRemoveCmd)
}
