package container

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/sirupsen/logrus"
)

// Container status constants
const (
	RUNNING = "running"
	STOP    = "stopped"
	EXIT    = "exited"
)

// Container directory constants
const (
	DefaultInfoLocation = "/var/run/mydocker/%s/"
	ConfigName          = "config.json"
	ContainerLogFile    = "container.log"

	// OverlayFS 相关目录
	ImagePath   = "/var/lib/mydocker/image/"
	OverlayRoot = "/var/lib/mydocker/overlay2/"
	LowerDir    = OverlayRoot + "%s/lower"
	UpperDir    = OverlayRoot + "%s/upper"
	WorkDir     = OverlayRoot + "%s/work"
	MergedDir   = OverlayRoot + "%s/merged"
)

// ContainerInfo 容器元信息
type ContainerInfo struct {
	Pid         string   `json:"pid"`         // 容器的init进程在宿主机上的 PID
	Id          string   `json:"id"`          // 容器Id
	Name        string   `json:"name"`        // 容器名
	Command     string   `json:"command"`     // 容器内init运行命令（字符串形式）
	Cmd         []string `json:"cmd"`         // 容器内init运行命令（数组形式，用于重启）
	CreatedTime string   `json:"createTime"`  // 创建时间
	Status      string   `json:"status"`      // 容器的状态
	Volume      string   `json:"volume"`      // 容器的数据卷
	PortMapping []string `json:"portmapping"` // 端口映射
	ImageName   string   `json:"imageName"`   // 镜像名称
	Env         []string `json:"env"`         // 环境变量
	NetworkName string   `json:"networkName"` // 网络名称
	IP          string   `json:"ip"`          // 容器 IP 地址
}

// RecordContainerInfo 记录容器信息
func RecordContainerInfo(containerPID int, commandArray []string, containerName, containerID, volume, imageName string, envSlice []string, networkName string) (*ContainerInfo, error) {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, " ")

	containerInfo := &ContainerInfo{
		Pid:         fmt.Sprintf("%d", containerPID),
		Id:          containerID,
		Name:        containerName,
		Command:     command,
		Cmd:         commandArray,
		CreatedTime: createTime,
		Status:      RUNNING,
		Volume:      volume,
		ImageName:   imageName,
		Env:         envSlice,
		NetworkName: networkName,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return nil, err
	}
	jsonStr := string(jsonBytes)

	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirURL, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirURL, err)
		return nil, err
	}
	fileName := dirURL + "/" + ConfigName
	file, err := os.Create(fileName)
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return nil, err
	}
	defer file.Close()

	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return nil, err
	}

	return containerInfo, nil
}

// DeleteContainerInfo 删除容器信息
func DeleteContainerInfo(containerName string) {
	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

// GetContainerInfoByName 根据容器名获取容器信息
func GetContainerInfoByName(containerName string) (*ContainerInfo, error) {
	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	configFileDir := dirURL + ConfigName
	contentBytes, err := os.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		log.Errorf("GetContainerInfoByName unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

// ListContainers 列出所有容器
func ListContainers() {
	dirURL := fmt.Sprintf(DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL)-1]

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")

	files, err := os.ReadDir(dirURL)
	if err != nil {
		if os.IsNotExist(err) {
			w.Flush()
			return
		}
		log.Errorf("Read dir %s error %v", dirURL, err)
		return
	}

	for _, file := range files {
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			tmpContainer.Id,
			tmpContainer.Name,
			tmpContainer.Pid,
			tmpContainer.Status,
			tmpContainer.Command,
			tmpContainer.CreatedTime)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
	}
}

func getContainerInfo(file os.DirEntry) (*ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := fmt.Sprintf(DefaultInfoLocation, containerName)
	configFileDir = configFileDir + ConfigName
	content, err := os.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("Read file %s error %v", configFileDir, err)
		return nil, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("Json unmarshal error %v", err)
		return nil, err
	}
	return &containerInfo, nil
}

// LogContainer 查看容器日志
func LogContainer(containerName string) {
	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	logFileLocation := dirURL + ContainerLogFile
	file, err := os.Open(logFileLocation)
	if err != nil {
		log.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}
	fmt.Fprint(os.Stdout, string(content))
}

// StopContainer 停止容器
func StopContainer(containerName string) error {
	info, err := GetContainerInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container %s info error: %v", containerName, err)
	}

	pidInt, err := GetPidFromPidStr(info.Pid)
	if err != nil {
		return fmt.Errorf("get pid from string error: %v", err)
	}

	// 发送 SIGTERM 信号
	if err := KillProcess(pidInt); err != nil {
		return fmt.Errorf("stop container %s error: %v", containerName, err)
	}

	// 更新容器状态
	info.Status = STOP
	info.Pid = ""
	if err := UpdateContainerInfo(info); err != nil {
		return fmt.Errorf("update container info error: %v", err)
	}

	return nil
}

// RemoveContainer 删除容器，force=true 时可强制删除运行中容器
func RemoveContainer(containerName string, force bool) error {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container %s info error: %v", containerName, err)
	}

	switch containerInfo.Status {
	case STOP:
		// 停止状态直接删除
	case RUNNING:
		if !force {
			return fmt.Errorf("couldn't remove running container [%s], use -f to force remove", containerName)
		}
		// 强制模式：先停止再删除
		if err := StopContainer(containerName); err != nil {
			return fmt.Errorf("stop container %s error: %v", containerName, err)
		}
	default:
		return fmt.Errorf("couldn't remove container, invalid status %s", containerInfo.Status)
	}

	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		return fmt.Errorf("remove file %s error: %v", dirURL, err)
	}

	DeleteWorkSpace(containerName, containerInfo.Volume)
	return nil
}

// InspectContainer 查看容器详细信息
func InspectContainer(containerName string) (*ContainerInfo, error) {
	return GetContainerInfoByName(containerName)
}

// UpdateContainerInfo 更新容器信息到磁盘
func UpdateContainerInfo(info *ContainerInfo) error {
	newContentBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}
	dirURL := fmt.Sprintf(DefaultInfoLocation, info.Name)
	configFilePath := filepath.Join(dirURL, ConfigName)
	if err := os.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		return fmt.Errorf("write file %s error: %v", configFilePath, err)
	}
	return nil
}
