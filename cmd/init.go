package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// initCmd 容器初始化命令（内部使用）
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init container process (internal use only)",
	Long:  "Initialize container process and run user's process in container. Do not call it outside.",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infof("init come on")
		return container.RunContainerInitProcess()
	},
}
