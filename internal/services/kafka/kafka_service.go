package kafka

import (
	"context"
	"time"

	myconfig "GoChatServer/internal/config"
	zlog "GoChatServer/pkg/zaplog"

	"github.com/segmentio/kafka-go"
)

var KafkaService = new(kafkaService)

var ctx = context.Background()

type kafkaService struct {
	// 读写流
	ChatWriter *kafka.Writer
	ChatReader *kafka.Reader
	// kafka连接
	KafkaConn *kafka.Conn
}

// 初始化kafka生产者和消费者客户端
func (k *kafkaService) KafkaInit() {
	kafkaConfig := myconfig.GetConfig().KafkaConfig
	// 初始化生产者
	k.ChatWriter = &kafka.Writer{
		Addr:                   kafka.TCP(kafkaConfig.HostPort),   // Kafka broker 的地址。kafka.TCP 是一个辅助函数，将 "host:port" 格式的字符串转换为内部地址类型。生产者通过这个地址连接到 Kafka 集群。
		Topic:                  kafkaConfig.ChatTopic,             // 当使用 Writer 发送消息时，从配置中读取了 ChatTopic。
		Balancer:               &kafka.Hash{},                     // 分区选择器（负载均衡器）。决定消息被发送到主题的哪个分区。&kafka.Hash{} 表示根据消息的 key 进行哈希计算，保证相同 key 的消息总是进入同一分区（从而保持顺序）。其他可选策略：RoundRobin（轮询）、LeastBytes（选择负载最小的分区）等。
		WriteTimeout:           kafkaConfig.Timeout * time.Second, // 写入操作的超时时间。如果发送消息在指定时间内未完成（包括等待 broker 确认），则会返回错误。这里从配置中读取 Timeout 并转换为 time.Duration。
		RequiredAcks:           kafka.RequireNone,                 // 生产者要求 broker 在写入消息后返回的确认级别。kafka.RequireNone 表示不等待任何确认（即“发后即忘”），性能最高但可能丢数据。其他级别：RequireOne（等待 leader 确认）、RequireAll（等待所有 ISR 副本确认），可靠性依次提高。
		AllowAutoTopicCreation: false,                             // 是否允许自动创建主题。如果设为 true，当生产者向一个不存在的主题发送消息时，Kafka 会自动创建该主题。这里设为 false 表示禁止自动创建，需要手动预先创建主题，避免误操作。
	}
	// 初始化消费者
	k.ChatReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaConfig.HostPort},    // Kafka broker 地址列表（切片）。这里只配置了一个地址，但可以配置多个以支持集群。消费者通过它们连接到 Kafka。
		Topic:          kafkaConfig.ChatTopic,             // 要消费的主题名称。与生产者使用的主题一致。
		CommitInterval: kafkaConfig.Timeout * time.Second, // 偏移量自动提交间隔。消费者会定期向 Kafka 报告当前已消费的消息位置（偏移量）。这样如果消费者重启，可以从上次提交的位置继续消费，避免重复或丢失。这里从配置中读取超时时间作为提交间隔。
		GroupID:        "chat",                            // 消费者组 ID。同一个组内的多个消费者共同消费一个主题，每条消息只会被组内的一个消费者处理。用于实现负载均衡和容错。这里组名为 "chat"，意味着所有使用该组 ID 的消费者实例将协同消费 ChatTopic 的消息。
		StartOffset:    kafka.LastOffset,                  // 当消费者没有已提交的偏移量时（例如第一次启动或偏移量过期），从哪个位置开始消费。kafka.LastOffset 表示从最新的消息开始消费（即只消费启动后产生的消息）。另一个常用选项是 kafka.FirstOffset，表示从最早的消息开始消费（重新处理历史数据）。
	})
}

// 停止kafka服务
func (k *kafkaService) KafkaClose() {
	if err := k.ChatWriter.Close(); err != nil {
		zlog.Error(err.Error())
	}
	if err := k.ChatReader.Close(); err != nil {
		zlog.Error(err.Error())
	}
}

// 创建topic
func (k *kafkaService) CreateTopic() {
	// 如果有topic则无需创建
	kafkaConfig := myconfig.GetConfig().KafkaConfig

	chatTopic := kafkaConfig.ChatTopic

	// 连接至任意kafka节点
	var err error
	k.KafkaConn, err = kafka.Dial("tcp", kafkaConfig.HostPort)
	if err != nil {
		zlog.Error(err.Error())
	}

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             chatTopic,
			NumPartitions:     kafkaConfig.Partition,
			ReplicationFactor: 1,
		},
	}

	// 创建topic
	if err = k.KafkaConn.CreateTopics(topicConfigs...); err != nil {
		zlog.Error(err.Error())
	}
}
