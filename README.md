## 描述

基于gin框架的分布式部署的聊天室

---
### 项目结构
```TEXT
GoChatServer/
├── api/                             # url
│   ├── user_info_controller.go      # 用户认证与资料
│   ├── message_controller.go        # 消息获取与文件上传
│   ├── group_info_controller.go     # 群组管理
│   ├── session_controller.go        # 会话操作
│   ├── user_contact_controller.go   # 联系人管理
│   ├── ws_controller.go             # WebSocket 端点
│   └── chatroom_controller.go       # 聊天室操作
├── internal/                        # 内部实现
│   ├── config/                      # 配置管理
│   │   └── config.go                # 基于 TOML 的配置结构
│   ├── model/                       # 数据库模型(实体)
│   │   ├── user_info.go             # 用户实体
│   │   ├── message.go               # 消息实体
│   │   ├── session.go               # 会话实体
│   │   ├── group_info.go            # 群组实体
│   │   ├── user_contact.go          # 联系人关系
│   │   └── contact_apply.go         # 好友申请
│   ├── service/                     # 业务/数据逻辑层
│   │   ├── chat/                    # WebSocket 服务器与客户端
│   │   │   ├── server.go            # 带消息路由的聊天服务器
│   │   │   ├── client.go            # 客户端连接管理
│   │   │   └── kafka_server.go      # 基于 Kafka 的消息处理
│   │   ├── gorm/                    # 数据库服务
│   │   ├── redis/                   # redis缓存操作
│   │   ├── kafka/                   # kafka消息队列操作
│   │   ├── sms/                     # aliyun oss短信验证
│   │   └── aes/                     # AES 加密服务
│   ├── dao/                         # 数据操作初始化
│   ├── dto/                         # 对前端的请求/响应 DTO
│   └── https_server/                # HTTP 服务器/路由设置
├── pkg/                             # 外部工具包
│   ├── constants/                   # 系统设置状态常量
│   ├── enum/                        # 实体状态枚举定义
│   ├── util/                        # 各种工具函数
│   ├── zaplog/                      # zaplog 日志封装
│   └── ssl/                         # TLS 处理
├── configs/                         # 全局配置文件
│   └── config.toml                  # 主配置
├── cmd/
│   └── server/            # 应用入口
│       └── main.go                  # 服务器启动
└── web/chat-server/                 # Vue3 前端
    ├── src/
    │   ├── views/                  # 页面组件
    │   ├── components/             # 可复用组件
    │   ├── router/                 # 路由定义
    │   └── store/                  # Vuex 状态管理
    └── package.json                # 前端依赖
```
---
### 后端技术栈
| 组件 | 技术 |
|------|------|
| 框架 | Gin (Go) |
| 数据库 ORM | GORM |
| 缓存 | GoRedis |
| 消息队列 | Kafka |
| 实时通讯 | WebSocket |
| 日志 | Zap |
| 加密 | 自定义 AES |
| 验证 | 阿里云短信 |

### 前端技术栈
| 组件 | 技术 |
|------|------|
| 框架 | Vue 3.2.13 |
| 状态管理 | Vuex 4.0.0 |
| 路由 | Vue Router 4.0.3 |
| UI 库 | Element Plus 2.9.0 |
| HTTP 客户端 | Axios 1.7.9 |
| 实时通讯 | WebSocket API |
| 包管理 | yarn |

---
### 快速开始
先修改/configs/config.toml的环境配置

#### 本地运行
##### 后端
```
$GOIM go mod tidy
$GOIM go run cmd/server/main.go
```

##### 前端
如果后端配置文件的port端口有修改, 则需要修改web/chat-server/src/main.js文件(配置backendUrl{后端的地址} & wsUrl{后端ws连接的地址})
```
$GOIM cd web/chat-server
$chat-server yarn install
$chat-server yarn serve
```
---
#### 功能实现
1、 后台开发者管理
赋予后台开发者管理权限，主要用于对用户群聊和用户角色进行管控。

开发者可以对用户群聊执行禁用、启用和删除操作，以确保群聊的正常秩序和合规性。同时，开发者还能根据实际需求，将普通用户设置为管理员，协助进行系统管理。

2、类似聊天软件的联系人体系

该体系模拟了常见聊天软件的联系人管理功能，为用户提供了丰富的社交互动选项。

用户可以自由添加或删除联系人，还能对联系人进行拉黑操作。

当用户想要添加新联系人时，可以发起申请，对方则有权选择同意或拒绝该申请。

3、 单聊与群聊

单聊和群聊是即时通讯系统的核心功能之一，其实现依赖于后端的聊天服务器。

无论是单聊消息还是群聊消息，都会被发送到后端服务器，由服务器负责将消息准确无误地转发给相应的接收方。

4、 多种消息类型的上传和下载（文本、文件、视频）

系统支持多种类型的消息交互，包括文本、文件和视频。用户可以方便地上传和下载这些不同类型的消息，满足多样化的沟通需求。

5、 Kafka

作为一个高效的消息队列系统，负责将客户端（client）发送的消息转发到服务器（server），确保消息的可靠传输和处理。

6、 语音视频通话

语音视频通话功能为用户提供了更加直观和实时的沟通方式。

用户可以发起通话邀请，对方可以选择接受或拒绝邀请。在通话过程中，任何一方都可以随时挂断通话，结束交流。

7、 WebSocket

WebSocket 为前端和客户端之间建立了实时、双向的通信连接。

它负责接收前端发送给客户端的消息，并将客户端的消息传递给服务器，实现了消息的高效传输和即时响应。

---
#### TODO:
- [ ] 编写dockerfile快速启动容器
- [ ] 通过email注册和登录
