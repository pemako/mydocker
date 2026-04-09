package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/pemako/mydocker/cmd"
)

func main() {
	// 配置日志格式
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// 执行根命令
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
