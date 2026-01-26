package cmd

import (
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
)

// commitCmd 提交容器为镜像
var commitCmd = &cobra.Command{
	Use:   "commit CONTAINER IMAGE_NAME",
	Short: "Commit a container into an image",
	Long:  "Create a new image from a container's changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("missing container name or image name")
		}
		containerName := args[0]
		imageName := args[1]
		commitContainer(containerName, imageName)
		return nil
	},
}

// commitContainer 提交容器为镜像
func commitContainer(containerName, imageName string) {
	mntURL := fmt.Sprintf(container.MntUrl, containerName)
	mntURL += "/*"
	imageTar := container.RootUrl + "/" + imageName + ".tar"
	log.Infof("mntURL = %s, imageTar = %s", mntURL, imageTar)

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		log.Errorf("Tar folder %s error %v", mntURL, err)
	}
}
