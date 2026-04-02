总体架构图
![alt text](pic/diagram(3).svg)
前端通过 WebSocket 和服务端实时连接
服务端处理消息时会写 MySQL
同时可能更新 Redis
如果是 kafka 模式，消息会先进入 Kafka Topic 再被处理

channel 模式启动图
![alt text](pic/diagram(4).svg)

channel 模式消息发送完整流程
前端发送 WebSocket 消息
 后端 WebSocket handler 收到消息
 调用 ChatServer.SendMessageToTransmit(message)
 消息进入 Server.Transmit
 Server.Start() 从 Transmit 中取出消息处理
 写数据库、更新 Redis、推送给目标用户

完整时序图:
![alt text](pic/diagram(5).svg)

内部结构图
![alt text](pic/diagram(6).svg)


kafka 模式完整流程图

kafka 模式启动图
![alt text](pic/diagram(7).svg)

kafka 模式消息发送完整流程
前端 WebSocket 发消息给服务器
 WebSocket handler 不直接处理业务
 而是把消息写入 Kafka Topic
 KafkaServer.Start() 内部 goroutine 从 Kafka 消费消息
 读到消息后：
    解析
    写数据库
    更新 Redis
    WebSocket 推送给目标用户
完整时序图
![alt text](pic/diagram(8).svg)


内部结构图
![alt text](pic/diagram(9).svg)


登录 / 退出流程图

登录流程图
![alt text](pic/diagram(10).svg)


退出流程图
![alt text](pic/diagram(11).svg)


#### 文字描述: 

```Text
开启websocket服务端
	概述
	传统http请求流程: 发起请求 → 服务器处理 → 返回响应 → 结束
	关于 websocket的消息流程
	/*
		websocket负责和浏览器保持实时连接(一种长连接协议) intro: 浏览器和服务器建立连接后，不断开, 双方都可以随时发消息
		( 	详细过程:
			1.用户上线时建立连接
			2.服务器把这个连接保存成 Client
			3.用户发消息时，服务器实时收到
			4.收到新消息后，服务器通过 WebSocket 实时推送给对方
			代码里这些都是 WebSocket 在线会话管理的一部分：
			
			Clients map[string]*Client
			Login chan *Client
			Logout chan *Client
			receiveClient.SendBack <- messageBack
			client.Conn.WriteMessage(...)
			
		)
		Kafka / channel：负责“服务器内部如何传递消息”
		MySQL/Gorm：负责消息持久化
		Redis：负责缓存消息列表

		1. client客户端中
		client struct {
			Conn     *websocket.Conn
			Uuid     string
			SendTo   chan []byte       // 给Server端
			SendBack chan *MessageBack // 给前端
		}
		实现了Read()方法for循环中(c.Conn.ReadMessage())持续读取前端websocket传来的消息
		通过json.Unmarshal(jsonMessage, &message)将前端传来的[]byte包装的json类型消息解析完
```

```
// main.go
if kafkaConfig.MessageMode == "channel" {
		go chat.ChatServer.Start()
} {...}
启动协程开启channel(普通内存通道版)的websocket服务端
    WebSocket → Go 内存通道 → 消息处理逻辑 → 推送给在线用户

			ChatServer(chat/server.go)内部有三个核心通道：
			Server struct {
				Clients  map[string]*Client  <- 保存着每个用户的连接 k: client.uuid v: *client
				mutex    *sync.Mutex
				Transmit chan []byte   // 转发通道
				Login    chan *Client  // 登录通道
				Logout   chan *Client  // 登出通道
			}
			用户建立 WebSocket 连接 即登录:
				简略流程:
				前端 WebSocket
					↓
				后端收到消息(创建一个 Client)
					↓
				ChatServer.Transmit
					↓
				ChatServer.Start() 消费
					↓
				反序列化
					↓
				写入MySQL
					↓
				查在线 Clients
					↓
				通过 WebSocket 推送给接收方/发送方
					↓
				更新 Redis

				详细流程:
				
					前端请求/wss时, 登录聊天服务器即将http升级到websocket
					GE.GET("/wss", api.WsLogin)
					// WsLogin wss登录 Get
					func WsLogin(c *gin.Context) {
						clientId := c.Query("client_id")
						if clientId == "" {
							zaplog.Error("clientId获取失败")
							c.JSON(http.StatusOK, gin.H{
								"code":    400,
								"message": "clientId获取失败",
							})
							return
						}
						chat.NewClientInit(c, clientId)
					}
					// NewClientInit 前端有登录消息时,则调用该函数
					func NewClientInit(c *gin.Context, clientId string) {
						kafkaConfig := config.GetConfig().KafkaConfig
						// 将一次 HTTP 请求升级为 WebSocket 协议，
						// 之后这个 TCP 连接就会一直保持为 WebSocket 长连接，直到客户端或服务器主动关闭。
						// 升级过程:
						// 	 upgrader.Upgrade(c.Writer, c.Request, nil) 通过解析请求头、验证协议版本、计算响应密钥、发送 101 状态码，
						// 	 最终将普通的 HTTP 连接升级为 WebSocket 连接，并返回一个可以直接读写 WebSocket 消息的 *websocket.Conn 对象。
						conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
						if err != nil {
							zaplog.Error(err.Error())
						}

						// 初始化用户连接信息
						client := &Client{
							// Conn (*websocket.Conn(封装了net.Conn),  TCP 连接的信息: 源 IP、源端口、目标 IP、目标端口) 具体起作用的是用户IP和用户端的http升级到websocket协议的端口号
							Conn:     conn,
							Uuid:     clientId,
							SendTo:   make(chan []byte, constants.CHANNEL_SIZE),
							SendBack: make(chan *MessageBack, constants.CHANNEL_SIZE),
						}
						// 选择channel或者kafka进行服务端的登录注册
						if kafkaConfig.MessageMode == "channel" {
							ChatServer.SendClientToLogin(client)
						} else {
							KafkaChatServer.SendClientToLogin(client)
						}
						// 这两行代码启动了两个新的并发任务，它们被调度到 Go 运行时的线程池中独立执行。
						// NewClientInit 函数本身只是一个初始化入口，它返回后，这两个 goroutine 依然在后台运行，各自维护着 client 对象的读写循环。
						// 这也是 WebSocket 长连接能够持续存在的关键。
						// 	函数返回 ≠ goroutine 结束。NewClientInit 返回后，Read 和 Write 继续运行，直到它们的内部循环条件失效。
						go client.Read()
						go client.Write()
						zaplog.Info("ws 连接成功")
					}

					ChatServer.Start() 处理登录
					在这里：
					NewClientInit调用SendClientToLogin先发送用户的TCP等信息到server.Login 登录通道中
					func (s *Server) SendClientToLogin(client *Client) {
						s.mutex.Lock()
						s.Login <- client
						s.mutex.Unlock()
					}
					// 在server.Start()开启的go协程中for循环里的case client := <-s.Login:接收到SendClientToLogin传来的s.Login信息并进行写入server.Clients中完成用户接入服务端的websocket
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

					// 前端用户消息发送流程
					从client的NewClientIni调用go Read()
					{
						Read() 的作用
						接收客户端消息：循环调用 c.Conn.ReadMessage() 阻塞读取客户端发来的 WebSocket 消息（文本或二进制）。
						解析与分发：{将收到的 JSON 消息解析为业务结构体（如 ChatMessageRequest）, 这一部分是在Server中处理写到Message数据表中的, client部分并没有对解析后的结构体进行任何处理}，然后根据配置（messageMode）进行处理：
							Channel 模式：优先将消息直接投递到 ChatServer.Transmit 通道（用于服务器内部转发）；若该通道满，则暂存到 c.SendTo 缓冲区；若缓冲区也满，则直接向客户端返回错误提示。
							Kafka 模式：将消息写入 Kafka 消息队列，实现异步解耦。
						错误处理：当读取失败（如连接关闭、网络中断）时，ReadMessage 返回错误，Read 方法退出，对应的 goroutine 终止，并触发连接清理。
						简单来说：Read 负责从 WebSocket 读取客户端消息，并将其导入后端处理流程（本地通道或 Kafka）。
					}
					// channel模式中, s.Transmit接收到消息后执行以下逻辑
					case data := <-s.Transmit: // 从通道里取出消息
					// client.Read()传来的
					{	将收到的 JSON 消息解析为业务结构体（如 ChatMessageRequest）
						var chatMessageReq request.ChatMessageRequest
						if err := json.Unmarshal(data, &chatMessageReq); err != nil {
							zaplog.Error(err.Error())
						}
						log.Println("原消息为：", data, "反序列化后为：", chatMessageReq)
					}
					{ // 然后将chatMessageReq解析后的结构体中的数据，按照数据类型分成不同的代码处理逻辑将消息存入后台数据库的Message表中,
					  // 并且判断消息发送对象 U-user G-Group 落库、推送给在线用户、更新 Redis
					  // 将消息塞入写回前端通道中(即广播){其sendClient和receiveClient如何确认, 结构体中的Clients[uuid]*Client、sendId和receiveId来确认, sendId, receiveId是从chatMessageReq结构体中获取的}
					  // sendClient.SendBack <- messageBack
					  // receiveClient.SendBack <- messageBack (其中接受者可能不在线, 但发送者肯定在线, 离线后通过存表，登录时只调用一次数据库操作完成离线消息的发送, 无论在线或离线都会存储消息)
					  if chatMessageReq.Type == message_type_enum.Text {...}
					}
					// 将发送者发来的数据进行一系列处理后receiveClient.SendBack <- messageBack, 在线的用户在NewInit之后一直在go client.Write()的协程中接收到从Server将消息塞入SendBack通道发来的数据
					// 从SendBack通道读取消息发送前端{即方法c.Conn.WriteMessage()}, 然后将数据库里存入的消息标记为已发送
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
					以上就是用户登录到发送消息到显示消息的全部流程
				
			用户退出流程:
			
				// WsLogout wss登出
				handler:
				func WsLogout(c *gin.Context) {
					var req request.WsLogoutRequest
					if err := c.BindJSON(&req); err != nil {
						zaplog.Error(err.Error())
						c.JSON(http.StatusOK, gin.H{
							"code":    500,
							"message": constants.SYSTEM_ERROR,
						})
						return
					}
					message, ret := chat.ClientLogout(req.OwnerId)
					JsonBack(c, message, ret, nil)
				}
				api:
				GE.POST("/user/wsLogout", api.WsLogout)
				当前端发送请求后的处理逻辑:
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
				发送给不同类型的websocket服务端的退出登录方法
				以channel为例:
				func (s *Server) SendClientToLogout(client *Client) {
					s.mutex.Lock()
					s.Logout <- client
					s.mutex.Unlock()
				}
				当把用户信息输入到Server.Logout通道中后
				case client := <-s.Logout: // 接收到消息, 调用RemoveClient方法(在Server.Clients表中删除该连接),并向退出通讯的client发送退出成功提醒
				{
					s.mutex.Lock()
					delete(s.Clients, client.Uuid)
					s.mutex.Unlock()
					zaplog.Info(fmt.Sprintf("user%v Logout\n", client.Uuid))
					if err := client.Conn.WriteMessage(websocket.TextMessage, []byte("Log out of chat")); err != nil {
						zaplog.Error(err.Error())
					}
				}
				func (s *Server) RemoveClient(uuid string) {
					s.mutex.Lock()
					delete(s.Clients, uuid)
					s.mutex.Unlock()
				}
```

```
// main,go
go chat.KafkaChatServer.Start() // 启动协程开启kafka的websocket服务端

    它是后端服务之间/服务内部的消息队列。(服务器内部的消息分发总线)
    你可以把它理解成一个“超强版消息中转站”：
        生产者把消息写进去
        消费者从里面读出来
        中间可以削峰、解耦、缓存、持久化
    Kafka 负责：
        接住聊天消息
        让消息先进入 topic
        由消费端统一处理
        再落库、再推送给在线用户


    他们的流程区别

    channel 模式
    浏览器
    ↓ WebSocket
    Go 服务
    ↓ chan
    消息处理逻辑
    ↓
    DB / Redis / 推送给在线用户

    kafka 模式
    浏览器
    ↓ WebSocket
    Go 服务
    ↓ Kafka Writer
    Kafka Topic
    ↓ Kafka Reader
    KafkaChatServer
    ↓
    DB / Redis / 推送给在线用户

	kafka简略流程
    前端 WebSocket
        ↓
    后端收到消息
        ↓
    Kafka Writer 写入 topic
        ↓
    Kafka Topic
        ↓
    Kafka Reader 读取
        ↓
    KafkaChatServer.Start() 消费
        ↓
    反序列化
        ↓
    写 MySQL
        ↓
    查在线 Clients
        ↓
    通过 WebSocket 推送给接收方
        ↓
    更新 Redis

    与channel的优势是kafka可以另起一台服务器去接收和发放这些数据(即分布式系统)，而channel(单机系统)只能在本机内存中去接收和发放当人流太多，内存容易爆满丢失信息


	// 分布式版（基于 Kafka 消息队列）

// kafka_server.go
	消息流转：移除了内部的 Transmit 通道，取而代之的是从 Kafka 中读取消息 (kafka.KafkaService.ChatReader.ReadMessage)。
	优势：支持集群部署。前端发送消息时，可以通过 HTTP 接口或其他方式先将消息投递到 Kafka。
	所有在线的 WebSocket 节点都会监听 Kafka。当节点从 Kafka 拿到消息后，会去自己的 Clients map 里找接收方是否在自己这台机器上，
	如果在就通过 WebSocket 发送。这样就打破了单机内存的限制。

	职责分离
    登录登出：在主程序的 for { select {...} } 中专门处理 Login 和 Logout。
    消息消费：单独开辟了一个后台协程 go func() { ... }() 用来死循环读取 Kafka 消息并处理业务逻辑。
    这样解耦后，消息处理的耗时不会阻塞新用户的登录和下线操作

	在 Start() 函数开头以及读取 Kafka 的子协程开头，都使用了 defer func() { if r := recover(); r != nil { ... } }()。
	这保证了即使某一条非法的 Kafka 消息导致了代码异常，服务也不会崩溃，只是在日志中记录错误后继续运行。

	在逻辑分支中增加了对文件类型 (message_type_enum.File) 的完整处理（包含发给单人 User 和 群组 Group 的逻辑），业务功能更加完整。

	Login, Logout 使用了无缓冲的 Channel：make(chan *Client)。
	在分布式下，单台机器的瞬时并发可能降低了，但无缓冲通道要求接收方和发送方必须同时准备好，严格来说，在高并发场景下依然建议加上缓冲大小。


	TODO:
	如果你的 Kafka 采用的是“发布/订阅 (广播)”模式（即所有节点都会收到同一条 Kafka 消息以寻找各自维护的连接）：
	那么当一条消息发来时，所有的服务器节点都会尝试将这条消息写入 MySQL 和 Redis，导致数据严重重复插入或 Redis 锁冲突。

	建议方案：在微服务架构下，写 MySQL 和 Redis 的操作应该放在 发送消息的 HTTP API 接口层。
	API 层将数据入库后，再把消息包装好推入 Kafka；
	而底层的 WebSocket 节点只负责做一件事：从 Kafka 取出消息 -> 匹配 k.Clients -> 推送给前端
	这样既能保证性能，又能避免分布式数据重复写入的问题。

```