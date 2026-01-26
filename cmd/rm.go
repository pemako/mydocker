package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// rmCmd 删除容器
var rmCmd = &cobra.Command{
	Use:   "rm CONTAINER",
	Short: "Remove a stopped container",
	Long:  "Remove a stopped container and clean up its resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		if err := container.RemoveContainer(containerName); err != nil {
			log.Errorf("Remove container error: %v", err)
			return err
		}
		return nil
	},
}
