package container

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	RootUrl             = "/root"
	MntUrl              = "/root/mnt/%s"
	WriteLayerUrl       = "/root/writeLayer/%s"
)

// ContainerInfo 容器元信息
type ContainerInfo struct {
	Pid         string   `json:"pid"`         // 容器的init进程在宿主机上的 PID
	Id          string   `json:"id"`          // 容器Id
	Name        string   `json:"name"`        // 容器名
	Command     string   `json:"command"`     // 容器内init运行命令
	CreatedTime string   `json:"createTime"`  // 创建时间
	Status      string   `json:"status"`      // 容器的状态
	Volume      string   `json:"volume"`      // 容器的数据卷
	PortMapping []string `json:"portmapping"` // 端口映射
}

// RecordContainerInfo 记录容器信息
func RecordContainerInfo(containerPID int, commandArray []string, containerName, containerID, volume string) (string, error) {
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := ""
	if len(commandArray) > 0 {
		command = commandArray[0]
		for i := 1; i < len(commandArray); i++ {
			command += " " + commandArray[i]
		}
	}

	containerInfo := &ContainerInfo{
		Pid:         fmt.Sprintf("%d", containerPID),
		Id:          containerID,
		Name:        containerName,
		Command:     command,
		CreatedTime: createTime,
		Status:      RUNNING,
		Volume:      volume,
	}

	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)

	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirUrl, err)
		return "", err
	}
	fileName := dirUrl + "/" + ConfigName
	file, err := os.Create(fileName)
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	defer file.Close()

	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
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
	contentBytes, err := ioutil.ReadFile(configFileDir)
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
	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		log.Errorf("Read dir %s error %v", dirURL, err)
		return
	}

	var containers []*ContainerInfo
	for _, file := range files {
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("Get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*ContainerInfo, error) {
	containerName := file.Name()
	configFileDir := fmt.Sprintf(DefaultInfoLocation, containerName)
	configFileDir = configFileDir + ConfigName
	content, err := ioutil.ReadFile(configFileDir)
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

	content, err := ioutil.ReadAll(file)
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
	newContentBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}

	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	configFilePath := filepath.Join(dirURL, ConfigName)
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		return fmt.Errorf("write file error: %v", err)
	}

	return nil
}

// RemoveContainer 删除容器
func RemoveContainer(containerName string) error {
	containerInfo, err := GetContainerInfoByName(containerName)
	if err != nil {
		return fmt.Errorf("get container %s info error: %v", containerName, err)
	}

	// 只能删除停止状态的容器
	if containerInfo.Status != STOP {
		return fmt.Errorf("couldn't remove running container")
	}

	dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		return fmt.Errorf("remove file %s error: %v", dirURL, err)
	}

	DeleteWorkSpace(containerInfo.Volume, containerName)
	return nil
}
