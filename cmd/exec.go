package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/pemako/mydocker/container"
	_ "github.com/pemako/mydocker/nsenter"
)

const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

// execCmd 在容器中执行命令
var execCmd = &cobra.Command{
	Use:   "exec CONTAINER COMMAND [ARG...]",
	Short: "Execute a command in a running container",
	Long:  "Execute a command in a running container",
	RunE: func(cmd *cobra.Command, args []string) error {
		// This is for nsenter callback
		if os.Getenv(ENV_EXEC_PID) != "" {
			log.Infof("pid callback pid %d", os.Getgid())
			return nil
		}

		if len(args) < 2 {
			return fmt.Errorf("missing container name or command")
		}

		containerName := args[0]
		commandArray := args[1:]
		ExecContainer(containerName, commandArray)
		return nil
	},
}

// ExecContainer 在运行的容器中执行命令
func ExecContainer(containerName string, comArray []string) {
	pid, err := GetContainerPidByName(containerName)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error %v", containerName, err)
		return
	}

	cmdStr := strings.Join(comArray, " ")
	log.Infof("container pid %s", pid)
	log.Infof("command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	containerEnvs := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerName, err)
	}
}

// GetContainerPidByName 获取容器的 PID
func GetContainerPidByName(containerName string) (string, error) {
	containerInfo, err := container.GetContainerInfoByName(containerName)
	if err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

// getEnvsByPid 从进程的环境变量文件获取环境变量
func getEnvsByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("Read file %s error %v", path, err)
		return nil
	}
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
