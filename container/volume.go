package container

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

// NewWorkSpace 创建容器工作空间（OverlayFS）
// 1. 创建 lower 层（解压镜像）
// 2. 创建 upper、work、merged 目录
// 3. 挂载 OverlayFS
// 4. 如果指定了 volume，挂载数据卷
func NewWorkSpace(containerID, imageName, volume string) error {
	if err := createLower(containerID, imageName); err != nil {
		return err
	}
	if err := createDirs(containerID); err != nil {
		return err
	}
	if err := mountOverlayFS(containerID); err != nil {
		return err
	}
	if volume != "" {
		hostPath, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed: %v", err)
			return err
		}
		mntPath := fmt.Sprintf(MergedDir, containerID)
		if err := mountVolume(mntPath, hostPath, containerPath); err != nil {
			return err
		}
	}
	return nil
}

// DeleteWorkSpace 删除容器工作空间
// 1. 先卸载 volume（必须在删除目录之前）
// 2. 卸载并删除 OverlayFS 目录
func DeleteWorkSpace(containerID, volume string) {
	if volume != "" {
		_, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed: %v", err)
		} else {
			mntPath := fmt.Sprintf(MergedDir, containerID)
			umountVolume(mntPath, containerPath)
		}
	}
	umountOverlayFS(containerID)
	deleteDirs(containerID)
}

// createLower 将镜像 tar 包解压到 lower 目录（只读层）
func createLower(containerID, imageName string) error {
	lowerPath := fmt.Sprintf(LowerDir, containerID)
	imagePath := ImagePath + imageName + ".tar"

	exist, err := PathExists(lowerPath)
	if err != nil {
		return fmt.Errorf("check lower dir %s error: %v", lowerPath, err)
	}
	if !exist {
		if err = os.MkdirAll(lowerPath, 0777); err != nil {
			return fmt.Errorf("mkdir lower dir %s error: %v", lowerPath, err)
		}
		if _, err = exec.Command("tar", "-xvf", imagePath, "-C", lowerPath).CombinedOutput(); err != nil {
			return fmt.Errorf("untar image %s error: %v", imagePath, err)
		}
	}
	return nil
}

// createDirs 创建 OverlayFS 需要的 merged、upper、work 目录
func createDirs(containerID string) error {
	dirs := []string{
		fmt.Sprintf(MergedDir, containerID),
		fmt.Sprintf(UpperDir, containerID),
		fmt.Sprintf(WorkDir, containerID),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("mkdir %s error: %v", dir, err)
		}
	}
	return nil
}

// mountOverlayFS 使用 OverlayFS 挂载容器文件系统
// mount -t overlay overlay -o lowerdir=...,upperdir=...,workdir=... merged
func mountOverlayFS(containerID string) error {
	lower := fmt.Sprintf(LowerDir, containerID)
	upper := fmt.Sprintf(UpperDir, containerID)
	work := fmt.Sprintf(WorkDir, containerID)
	merged := fmt.Sprintf(MergedDir, containerID)

	dirs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, merged)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mount overlayfs error: %v", err)
	}
	return nil
}

// umountOverlayFS 卸载 OverlayFS
func umountOverlayFS(containerID string) {
	merged := fmt.Sprintf(MergedDir, containerID)
	cmd := exec.Command("umount", merged)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount overlayfs %s error: %v", merged, err)
	}
}

// deleteDirs 删除 OverlayFS 所有目录（lower/upper/work/merged 及根目录）
func deleteDirs(containerID string) {
	dirs := []string{
		fmt.Sprintf(MergedDir, containerID),
		fmt.Sprintf(UpperDir, containerID),
		fmt.Sprintf(WorkDir, containerID),
		fmt.Sprintf(LowerDir, containerID),
		OverlayRoot + containerID,
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Errorf("remove dir %s error: %v", dir, err)
		}
	}
}

// mountVolume 使用 bind mount 挂载数据卷
func mountVolume(mntPath, hostPath, containerPath string) error {
	if err := os.MkdirAll(hostPath, 0777); err != nil {
		log.Infof("mkdir host dir %s error: %v", hostPath, err)
	}
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.MkdirAll(containerPathInHost, 0777); err != nil {
		log.Infof("mkdir container dir %s error: %v", containerPathInHost, err)
	}
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mount volume failed: %v", err)
	}
	return nil
}

// umountVolume 卸载数据卷
func umountVolume(mntPath, containerPath string) {
	containerPathInHost := path.Join(mntPath, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("umount volume %s error: %v", containerPathInHost, err)
	}
}

// volumeExtract 解析 volume 参数（格式: hostPath:containerPath）
func volumeExtract(volume string) (sourcePath, destPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid volume [%s], must be hostPath:containerPath", volume)
	}
	return parts[0], parts[1], nil
}
