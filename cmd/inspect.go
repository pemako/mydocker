package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// inspectCmd 查看容器详细信息
var inspectCmd = &cobra.Command{
	Use:   "inspect CONTAINER",
	Short: "Display detailed information on a container",
	Long:  "Display detailed information on a container in JSON format",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := args[0]
		info, err := container.InspectContainer(containerName)
		if err != nil {
			log.Errorf("Inspect container error: %v", err)
			return err
		}

		data, err := json.MarshalIndent(info, "", "    ")
		if err != nil {
			return fmt.Errorf("json marshal error: %v", err)
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	},
}
