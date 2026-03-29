package chat

import (
	"GoChatServer/internal/config"
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/message/message_status_enum"
	"GoChatServer/pkg/zaplog"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	mykafka "GoChatServer/internal/services/kafka"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

type MessageBack struct {
	Message []byte
	Uuid    string
}

type Client struct {
	Conn     *websocket.Conn
	Uuid     string
	SendTo   chan []byte       // 给Server端
	SendBack chan *MessageBack // 给前端
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
	// 检查连接的Origin头
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var ctx = context.Background()

var messageMode = config.GetConfig().KafkaConfig.MessageMode

// 读取websocket消息并发送send通道
func (c *Client) Read() {
	zaplog.Info("ws read goroutine start!")
	for {
		// TODO 阻塞有一定隐患，因为下面要处理缓冲的逻辑，但是可以先不做优化，问题不大
		// WebSocket 收到前端消息
		_, jsonMessage, err := c.Conn.ReadMessage() // 阻塞状态
		if err != nil {
			zaplog.Error(err.Error())
			return // 直接断开websocket连接
		} else {
			var message = request.ChatMessageRequest{}
			if err := json.Unmarshal(jsonMessage, &message); err != nil {
				zaplog.Error(err.Error())
			}
			log.Println("接收到的消息:", message)

			if messageMode == "channel" {
				// 直接丢进 ChatServer.Transmit
				// 如果Server的转发channel没满, 先把sendTo中的给transmit
				for len(ChatServer.Transmit) < constants.CHANNEL_SIZE && len(c.SendTo) > 0 {
					SendToMessage := <-c.SendTo
					ChatServer.SendMessageToTransmit(SendToMessage)
				}

				// 如果Server没满, SendTo空了, 直接给Server Transmit
				if len(ChatServer.Transmit) < constants.CHANNEL_SIZE {
					ChatServer.SendMessageToTransmit(jsonMessage)
				} else if len(c.SendTo) < constants.CHANNEL_SIZE {
					// 如果Server满了, 直接塞给SendTo
					c.SendTo <- jsonMessage
				} else {
					// 否则考虑加宽 channel size, 或者使用kafka
					if err := c.Conn.WriteMessage(websocket.TextMessage, []byte("由于同一时间过多用户发送消息，消息发送失败，请稍后重试")); err != nil {
						zaplog.Error(err.Error())
					}
				}
			} else {
				if err := mykafka.KafkaService.ChatWriter.WriteMessages(ctx, kafka.Message{
					Key:   []byte(strconv.Itoa(config.GetConfig().KafkaConfig.Partition)),
					Value: jsonMessage,
				}); err != nil {
					zaplog.Error(err.Error())
				}
				zaplog.Info("已发送消息: " + string(jsonMessage))
			}
		}
	}
}

// 从send通道读取消息发送给websocket
func (c *Client) Write() {
	zaplog.Info("websocket write goroutine start")
	for messageBack := range c.SendBack { // 阻塞态
		// 通过websocket发送消息
		err := c.Conn.WriteMessage(websocket.TextMessage, messageBack.Message)
		if err != nil {
			zaplog.Error(err.Error())
			return // 直接断开websocket连接
		}
		log.Println("已发送消息:", messageBack.Message)
		if res := mysql.GormDB.Model(&model.Message{}).Where("uuid = ?", messageBack.Uuid).Update("status", message_status_enum.Sent); res.Error != nil {
			zaplog.Error(res.Error.Error())
		}
	}
}

// NewClientInit 前端有登录消息时,则调用该函数
func NewClientInit(c *gin.Context, clientId string) {
	kafkaConfig := config.GetConfig().KafkaConfig
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zaplog.Error(err.Error())
	}

	client := &Client{
		Conn:     conn,
		Uuid:     clientId,
		SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
		SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
	}
	if kafkaConfig.MessageMode == "channel" {
		ChatServer.SendClientToLogin(client)
	} else {
		KafkaChatServer.SendClientToLogin(client)
	}
	go client.Read()
	go client.Write()
	zaplog.Info("ws 连接成功")
}

// ClientLogout	当接收到前端发来的登出请求. 则调用该函数
func ClientLogout(clientId string) (string, int) {
	kafkaConfig := config.GetConfig().KafkaConfig
	client := ChatServer.Clients[clientId]
	if client != nil {
		if kafkaConfig.MessageMode == "channel" {
			ChatServer.SendClientToLogout(client)
		} else {
			KafkaChatServer.SendClientToLogout(client)
		}
		if err := client.Conn.Close(); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		close(client.SendTo)
		close(client.SendBack)
	}
	return "退出成功", 0
}
