package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/pemako/mydocker/cmd"
)

const usage = `mydocker is a simple container runtime implementation.
The purpose of this project is to learn how docker works and how to write a docker by ourselves
Enjoy it, just for fun.`

func main() {
	// 配置日志格式
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// 执行根命令
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
