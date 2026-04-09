package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
	nw "github.com/pemako/mydocker/network"
)

// rmCmd 删除容器
var rmCmd = &cobra.Command{
	Use:   "rm CONTAINER",
	Short: "Remove a container",
	Long:  "Remove a stopped container and clean up its resources. Use -f to force remove a running container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		force, _ := cmd.Flags().GetBool("force")

		// 获取容器信息（用于网络清理）
		info, err := container.GetContainerInfoByName(containerName)
		if err != nil {
			log.Errorf("Get container info error: %v", err)
			return err
		}

		if err := container.RemoveContainer(containerName, force); err != nil {
			log.Errorf("Remove container error: %v", err)
			return err
		}

		// 清理网络资源
		if info.NetworkName != "" {
			if err := nw.Disconnect(info.NetworkName, info); err != nil {
				log.Errorf("Disconnect network error: %v", err)
			}
		}
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "Force remove a running container")
}
