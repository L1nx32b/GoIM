总体架构图
![alt text](diagram(3).svg)
前端通过 WebSocket 和服务端实时连接
服务端处理消息时会写 MySQL
同时可能更新 Redis
如果是 kafka 模式，消息会先进入 Kafka Topic 再被处理


channel 模式启动图
![alt text](diagram(4).svg)

channel 模式消息发送完整流程
前端发送 WebSocket 消息
 后端 WebSocket handler 收到消息
 调用 ChatServer.SendMessageToTransmit(message)
 消息进入 Server.Transmit
 Server.Start() 从 Transmit 中取出消息处理
 写数据库、更新 Redis、推送给目标用户

完整时序图:
![alt text](diagram(5).svg)

内部结构图
![alt text](diagram(6).svg)


kafka 模式完整流程图

kafka 模式启动图
![alt text](diagram(7).svg)

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
![alt text](diagram(8).svg)


内部结构图
![alt text](diagram(9).svg)


登录 / 退出流程图

登录流程图
![alt text](diagram(10).svg)


退出流程图
![alt text](diagram(11).svg)

