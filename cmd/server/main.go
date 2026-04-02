package main

import (
	"GoChatServer/internal/config"
	"GoChatServer/internal/https_server"
	"GoChatServer/internal/services/chat"
	"GoChatServer/internal/services/kafka"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/zaplog"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	kafkaConfig := conf.KafkaConfig // kafka配置

	if kafkaConfig.MessageMode == "kafka" {
		// 只有当配置是 "kafka" 时，才初始化 Kafka 相关组件：
		// ChatWriter
		// ChatReader
		kafka.KafkaService.KafkaInit()
	}

	if kafkaConfig.MessageMode == "channel" {
		go chat.ChatServer.Start()
	} else {
		go chat.KafkaChatServer.Start() // 启动协程开启kafka的websocket服务端
	}
	// 在 Go gin.Run(）是一个阻塞调用，它会一直运行直到服务器关闭或发生致命错误。
	// 使用 go func() 将其放入独立的 goroutine 中，目的是避免阻塞主 goroutine，让主程序可以继续执行其他任务。
	go func() {
		// devserver
		if err := https_server.GE.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			zaplog.Fatal("server running fault")
			return
		}
	}()

	// 设置信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号, 释放后面的资源
	<-quit

	if kafkaConfig.MessageMode == "kafka" {
		kafka.KafkaService.KafkaClose()
	}

	chat.ChatServer.Close()

	zaplog.Info("关闭服务器...")

	// 删除所有Redis键
	if err := myredis.DeleteAllRedisKey(); err != nil {
		zaplog.Error(err.Error())
	} else {
		zaplog.Info("所有Redis键已删除")
	}

	zaplog.Info("服务器已关闭")
}
