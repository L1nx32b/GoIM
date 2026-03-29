package chat

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/message/message_status_enum"
	"GoChatServer/pkg/enum/message/message_type_enum"
	"GoChatServer/pkg/util/random"
	"GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

// 单机版（基于内存 Channel）

/*
	消息流转：完全依靠 Go 内部的 channel（特别是 Transmit 通道）。
	客户端发送的消息被放入 Transmit，然后在 Start() 的 select 循环中被消费并分发给目标客户端。
	局限性：这是一个纯单机应用。
	如果部署了多个服务端节点（比如 Node A 和 Node B），连接在 Node A 的用户无法直接给连接在 Node B 的用户发送消息，
	因为它们的 Transmit 通道和 Clients map 是相互隔离的内存数据。

	单点阻塞式监听
    所有的操作（登录、登出、消息转发）都在一个大的 for { select {...} } 中同步进行。
    如果某一条消息的业务逻辑（比如写 MySQL/Redis）处理过慢，会阻塞整个循环，导致其他用户的登录、登出或消息转发延迟。

	没有做任何 panic 捕获。如果处理某条消息时发生空指针或严重错误导致 panic，整个 WebSocket 服务端进程将会崩溃退出。

	只处理了文本 (message_type_enum.Text) 和音视频 (message_type_enum.AudioOrVideo) 两种消息类型。

	Login, Logout, Transmit 全部使用了带缓冲的 Channel：make(chan *Client, constants.CHANNEL_SIZE)。
	这可以防止在高并发登录/发消息时短暂的拥塞导致协程死锁或阻塞。
*/

type Server struct {
	Clients  map[string]*Client
	mutex    *sync.Mutex
	Transmit chan []byte  // 转发通道
	Login    chan *Client // 登录通道
	Logout   chan *Client // 退出通道
}

var ChatServer *Server

func init() {
	if ChatServer == nil {
		ChatServer = &Server{
			Clients:  make(map[string]*Client),
			mutex:    &sync.Mutex{},
			Transmit: make(chan []byte, constants.CHANNEL_SIZE),
			Login:    make(chan *Client, constants.CHANNEL_SIZE),
			Logout:   make(chan *Client, constants.CHANNEL_SIZE),
		}
	}
}

func normalizePath(path string) string {
	// 查找 /static/ 位置
	if path == "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png" {
		return path
	}

	staticIndex := strings.Index(path, "/static")
	if staticIndex < 0 {
		log.Println(path)
		zaplog.Error("路径不合法")
	}

	// 返回从 "/static/" 开始的部分
	return path[staticIndex:]
}

// Start启动函数, Server端用主进程起, Client用协程起
func (s *Server) Start() {
	defer func() {
		close(s.Transmit)
		close(s.Login)
		close(s.Logout)
	}()
	for {
		select {
		case client := <-s.Login:
			{
				s.mutex.Lock()
				s.Clients[client.Uuid] = client
				s.mutex.Unlock()
				zaplog.Debug(fmt.Sprintf("Welcome ChatServer! user: %v\n", client.Uuid))
				err := client.Conn.WriteMessage(websocket.TextMessage, []byte("Welcome ChatServer"))
				if err != nil {
					zaplog.Error(err.Error())
				}
			}
		case client := <-s.Logout:
			{
				s.mutex.Lock()
				delete(s.Clients, client.Uuid)
				s.mutex.Unlock()
				zaplog.Info(fmt.Sprintf("user%v Logout\n", client.Uuid))
				if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("Log out of chat")); err != nil {
					zaplog.Error(err.Error())
				}
			}
		case data := <-s.Transmit: // 从通道里取出消息
			{
				var chatMessageReq request.ChatMessageRequest
				if err := json.Unmarshal(data, &chatMessageReq); err != nil {
					zaplog.Error(err.Error())
				}

				log.Println("原消息为：", data, "反序列化后为：", chatMessageReq)
				// 如果是TEXT信息
				if chatMessageReq.Type == message_type_enum.Text {
					// 存放Message
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    chatMessageReq.Content,
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "0B",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容, 防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := mysql.GormDB.Create(&message); res.Error != nil {
						zaplog.Error(res.Error.Error())
					}
					// 判断消息发送对象 U-user G-Group 落库、推送给在线用户、更新 Redis
					if message.ReceiveId[0] == 'U' {
						// 如果能找到ReceiveId,说明在线,可以发送, 否则存入表中跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zaplog.Error(err.Error())
						}
						log.Println("返回的消息:", messageRsp, "序列化后:", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}

						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 因为send_id肯定在线，所以这里在后端进行在线回显message，其实优化的话前端可以直接回显
						// 问题在于前后端的req和rsp结构不同，前端存储message的messageList不能存req，只能存rsp
						// 所以这里后端进行回显，前端不回显
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							// 如果redis缓存中有数据
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zaplog.Error(err.Error())
							}

							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zaplog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zaplog.Error(err.Error())
							}
						} else {
							// 如果redis出错
							if !errors.Is(err, redis.Nil) {
								zaplog.Error(err.Error())
							}
						}

					} else if message.ReceiveId[0] == 'G' {
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}

						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zaplog.Error(err.Error())
						}
						log.Println("返回消息:", messageRsp, "序列化后：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}

						var group model.GroupInfo
						if res := mysql.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zaplog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zaplog.Error(err.Error())
						}

						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zaplog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zaplog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zaplog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zaplog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.File {
					// 存message
					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        chatMessageReq.Url,
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   chatMessageReq.FileSize,
						FileType:   chatMessageReq.FileType,
						FileName:   chatMessageReq.FileName,
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     "",
					}
					// 对SendAvatar去除前面/static之前的所有内容，防止ip前缀引入
					message.SendAvatar = normalizePath(message.SendAvatar)
					if res := mysql.GormDB.Create(&message); res.Error != nil {
						zaplog.Error(res.Error.Error())
					}

					if message.ReceiveId[0] == 'U' { // 发送给User
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.GetMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zaplog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 因为send_id肯定在线，所以这里在后端进行在线回显message，其实优化的话前端可以直接回显
						// 问题在于前后端的req和rsp结构不同，前端存储message的messageList不能存req，只能存rsp
						// 所以这里后端进行回显，前端不回显
						sendClient := s.Clients[message.SendId]
						sendClient.SendBack <- messageBack
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("message_list_" + message.SendId + "_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zaplog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zaplog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("message_list_"+message.SendId+"_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zaplog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zaplog.Error(err.Error())
							}
						}
					} else { // 发送给群聊
						messageRsp := respond.GetGroupMessageListRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: chatMessageReq.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zaplog.Error(err.Error())
						}
						log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						var group model.GroupInfo
						if res := mysql.GormDB.Where("uuid = ?", message.ReceiveId).First(&group); res.Error != nil {
							zaplog.Error(res.Error.Error())
						}
						var members []string
						if err := json.Unmarshal(group.Members, &members); err != nil {
							zaplog.Error(err.Error())
						}
						s.mutex.Lock()
						for _, member := range members {
							if member != message.SendId {
								if receiveClient, ok := s.Clients[member]; ok {
									receiveClient.SendBack <- messageBack
								}
							} else {
								sendClient := s.Clients[message.SendId]
								sendClient.SendBack <- messageBack
							}
						}
						s.mutex.Unlock()

						// redis
						var rspString string
						rspString, err = myredis.GetKeyNilIsErr("group_messagelist_" + message.ReceiveId)
						if err == nil {
							var rsp []respond.GetGroupMessageListRespond
							if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
								zaplog.Error(err.Error())
							}
							rsp = append(rsp, messageRsp)
							rspByte, err := json.Marshal(rsp)
							if err != nil {
								zaplog.Error(err.Error())
							}
							if err := myredis.SetKeyEx("group_messagelist_"+message.ReceiveId, string(rspByte), time.Minute*constants.REDIS_TIMEOUT); err != nil {
								zaplog.Error(err.Error())
							}
						} else {
							if !errors.Is(err, redis.Nil) {
								zaplog.Error(err.Error())
							}
						}
					}
				} else if chatMessageReq.Type == message_type_enum.AudioOrVideo {
					var avData request.AVData
					if err := json.Unmarshal([]byte(chatMessageReq.AVdata), &avData); err != nil {
						zaplog.Error(err.Error())
					}
					log.Println(avData)

					message := model.Message{
						Uuid:       fmt.Sprintf("M%s", random.GetNowAndLenRandomString(11)),
						SessionId:  chatMessageReq.SessionId,
						Type:       chatMessageReq.Type,
						Content:    "",
						Url:        "",
						SendId:     chatMessageReq.SendId,
						SendName:   chatMessageReq.SendName,
						SendAvatar: chatMessageReq.SendAvatar,
						ReceiveId:  chatMessageReq.ReceiveId,
						FileSize:   "",
						FileType:   "",
						FileName:   "",
						Status:     message_status_enum.Unsent,
						CreatedAt:  time.Now(),
						AVdata:     chatMessageReq.AVdata,
					}
					if avData.MessageId == "PROXY" && (avData.Type == "start_call" || avData.Type == "receive_call" || avData.Type == "reject_call") {
						// 存message
						// 对SendAvatar 去除前面/static之前的所有内容, 放置ip前缀引入
						message.SendAvatar = normalizePath(message.SendAvatar)
						if res := mysql.GormDB.Create(&message); res.Error != nil {
							zaplog.Error(res.Error.Error())
						}
					}
					// 判断消息发送对象 U-user G-Group
					if chatMessageReq.ReceiveId[0] == 'U' {
						// 如果能找到ReceiveId，说明在线，可以发送，否则存表后跳过
						// 因为在线的时候是通过websocket更新消息记录的，离线后通过存表，登录时只调用一次数据库操作
						// 切换chat对象后，前端的messageList也会改变，获取messageList从第二次就是从redis中获取
						messageRsp := respond.AVMessageRespond{
							SendId:     message.SendId,
							SendName:   message.SendName,
							SendAvatar: message.SendAvatar,
							ReceiveId:  message.ReceiveId,
							Type:       message.Type,
							Content:    message.Content,
							Url:        message.Url,
							FileSize:   message.FileSize,
							FileName:   message.FileName,
							FileType:   message.FileType,
							CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
							AVdata:     message.AVdata,
						}
						jsonMessage, err := json.Marshal(messageRsp)
						if err != nil {
							zaplog.Error(err.Error())
						}
						// log.Println("返回的消息为：", messageRsp, "序列化后为：", jsonMessage)
						log.Println("返回的消息为：", messageRsp)
						var messageBack = &MessageBack{
							Message: jsonMessage,
							Uuid:    message.Uuid,
						}
						s.mutex.Lock()
						if receiveClient, ok := s.Clients[message.ReceiveId]; ok {
							//messageBack.Message = jsonMessage
							//messageBack.Uuid = message.Uuid
							receiveClient.SendBack <- messageBack // 向client.Send发送
						}
						// 通话这不能回显，发回去的话就会出现两个start_call。
						//sendClient := s.Clients[message.SendId]
						//sendClient.SendBack <- messageBack
						s.mutex.Unlock()
					}
				}

			}
		}
	}
}

func (s *Server) Close() {
	close(s.Login)
	close(s.Logout)
	close(s.Transmit)
}

func (s *Server) SendClientToLogin(client *Client) {
	s.mutex.Lock()
	s.Login <- client
	s.mutex.Unlock()
}

func (s *Server) SendClientToLogout(client *Client) {
	s.mutex.Lock()
	s.Logout <- client
	s.mutex.Unlock()
}

func (s *Server) SendMessageToTransmit(message []byte) {
	s.mutex.Lock()
	s.Transmit <- message
	s.mutex.Unlock()
}

func (s *Server) RemoveClient(uuid string) {
	s.mutex.Lock()
	delete(s.Clients, uuid)
	s.mutex.Unlock()
}
