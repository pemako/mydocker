package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/cgroups/subsystems"
	"github.com/pemako/mydocker/container"
)

// restartCmd 重启容器
var restartCmd = &cobra.Command{
	Use:   "restart CONTAINER",
	Short: "Restart a container",
	Long:  "Stop a running container (if needed) and start it again with the same configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		return restartContainer(containerName)
	},
}

func restartContainer(containerName string) error {
	info, err := container.GetContainerInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container %s info error: %v", containerName, err)
	}

	// 如果容器正在运行，先停止
	if info.Status == container.RUNNING {
		log.Infof("Container %s is running, stopping first", containerName)
		if err := container.StopContainer(containerName); err != nil {
			return fmt.Errorf("stop container %s error: %v", containerName, err)
		}
		// 重新读取更新后的信息
		info, err = container.GetContainerInfoByName(containerName)
		if err != nil {
			return fmt.Errorf("get container %s info after stop error: %v", containerName, err)
		}
	}

	if info.Status != container.STOP {
		return fmt.Errorf("container %s is not in stopped state (status: %s)", containerName, info.Status)
	}

	if info.ImageName == "" {
		return fmt.Errorf("container %s has no image info, cannot restart", containerName)
	}

	log.Infof("Restarting container %s with image %s", containerName, info.ImageName)

	// 以后台模式重新运行容器（使用保存的配置）
	Run(
		false, // 重启时使用非tty后台模式
		info.Cmd,
		&subsystems.ResourceConfig{},
		containerName,
		info.Volume,
		info.ImageName,
		info.Env,
		info.NetworkName,
		info.PortMapping,
	)
	return nil
}
