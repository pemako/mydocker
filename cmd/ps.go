package cmd

import (
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// psCmd 列出所有容器
var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List all containers",
	Long:  "List all the containers with their status and information",
	RunE: func(cmd *cobra.Command, args []string) error {
		container.ListContainers()
		return nil
	},
}
