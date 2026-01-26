package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/cgroups"
	"github.com/pemako/mydocker/cgroups/subsystems"
	"github.com/pemako/mydocker/container"
)

// runCmd 运行容器命令
var runCmd = &cobra.Command{
	Use:   "run [flags] IMAGE [COMMAND] [ARG...]",
	Short: "Create a container with namespace and cgroups limit",
	Long:  "Create and run a new container with the specified image and command",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing container image")
		}

		tty, _ := cmd.Flags().GetBool("ti")
		detach, _ := cmd.Flags().GetBool("d")

		if tty && detach {
			return fmt.Errorf("ti and d parameter cannot both be provided")
		}

		memory, _ := cmd.Flags().GetString("m")
		cpushare, _ := cmd.Flags().GetString("cpushare")
		cpuset, _ := cmd.Flags().GetString("cpuset")
		containerName, _ := cmd.Flags().GetString("name")
		volume, _ := cmd.Flags().GetString("v")
		network, _ := cmd.Flags().GetString("net")
		envSlice, _ := cmd.Flags().GetStringSlice("e")
		portMapping, _ := cmd.Flags().GetStringSlice("p")

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: memory,
			CpuSet:      cpuset,
			CpuShare:    cpushare,
		}

		imageName := args[0]
		var commandArray []string
		if len(args) > 1 {
			commandArray = args[1:]
		}

		Run(tty, commandArray, resConf, containerName, volume, imageName, envSlice, network, portMapping)
		return nil
	},
}

func init() {
	// run 命令的标志
	runCmd.Flags().BoolP("ti", "t", false, "Enable tty")
	runCmd.Flags().BoolP("d", "d", false, "Detach container")
	runCmd.Flags().StringP("m", "m", "", "Memory limit (e.g., 100m, 1g)")
	runCmd.Flags().String("cpushare", "", "CPU share limit (default 1024)")
	runCmd.Flags().String("cpuset", "", "CPUset limit (e.g., 0-2, 0,1)")
	runCmd.Flags().String("name", "", "Container name")
	runCmd.Flags().StringP("v", "v", "", "Volume mapping (e.g., /host/path:/container/path)")
	runCmd.Flags().String("net", "", "Container network")
	runCmd.Flags().StringSliceP("e", "e", []string{}, "Set environment variables")
	runCmd.Flags().StringSliceP("p", "p", []string{}, "Port mapping")
}

// Run 运行容器
func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, containerName, volume, imageName string,
	envSlice []string, network string, portMapping []string) {

	containerID := randStringBytes(10)
	if containerName == "" {
		containerName = containerID
	}

	parent, writePipe := container.NewParentProcess(tty, containerName, volume, imageName, envSlice)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
		return
	}

	// Record container info
	if _, err := container.RecordContainerInfo(parent.Process.Pid, comArray, containerName, containerID, volume); err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}

	// use mydocker-cgroup as cgroup name
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	sendInitCommand(comArray, writePipe)

	if tty {
		parent.Wait()
		container.DeleteWorkSpace(volume, containerName)
		container.DeleteContainerInfo(containerName)
	}
}

// sendInitCommand 通过管道发送初始化命令
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// randStringBytes 生成随机字符串作为容器 ID
func randStringBytes(n int) string {
	letterBytes := "0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
