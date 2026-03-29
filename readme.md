```TEXT
// cofing 配置
mainConfig	服务器基本设置	主机、端口、应用名称
mysqlConfig	数据库连接	主机、端口、凭据、数据库名称
redisConfig	缓存配置	主机、端口、密码、数据库索引
kafkaConfig	消息队列设置	模式 (channel/kafka)、主题、分区
authCodeConfig	短信验证	阿里云密钥、模板代码
staticSrcConfig	文件存储路径	头像和文件存储位置
logConfig	日志配置	日志文件路径
```

代码树
```TEXT
GoChatServer/
├── api/v1/  # url
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
│   ├── zlog/                        # zaplog 日志封装
│   └── ssl/                         # TLS 处理
├── configs/                         # 全局配置文件
│   └── config.toml                  # 主配置
├── cmd/
│   └── kama_chat_server/            # 应用入口
│       └── main.go                  # 服务器初始化
└── web/chat-server/                 # Vue3 前端
    ├── src/
    │   ├── views/                  # 页面组件
    │   ├── components/             # 可复用组件
    │   ├── router/                 # 路由定义
    │   └── store/                  # Vuex 状态管理
    └── package.json                # 前端依赖
```
