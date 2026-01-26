package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// logsCmd 查看容器日志
var logsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "Print logs of a container",
	Long:  "Print the logs of a container",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		container.LogContainer(containerName)
		return nil
	},
}
