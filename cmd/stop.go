package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// stopCmd 停止容器
var stopCmd = &cobra.Command{
	Use:   "stop CONTAINER",
	Short: "Stop a running container",
	Long:  "Stop a running container by sending SIGTERM signal",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		if err := container.StopContainer(containerName); err != nil {
			log.Errorf("Stop container error: %v", err)
			return err
		}
		return nil
	},
}
