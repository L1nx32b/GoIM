package config

// 全局配置
// 配置文件: ./configs/config.toml中
import (
	"log"
	"time"

	"github.com/BurntSushi/toml"
)

// 主要配置
type MainConfig struct {
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}

// Mysql数据库配置
type MysqlConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	Password     string `toml:"password"`
	DatabaseName string `toml:"databaseName"`
}

// Redis缓存配置
type RedisConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Password string `toml:"password"`
	Db       int    `toml:"db"`
}

// 短信认证配置
type AuthCodeConfig struct {
	AccessKeyID     string `toml:"accessKeyID"`
	AccessKeySecret string `toml:"accessKeySecret"`
	SignName        string `toml:"signName"`
	TemplateCode    string `toml:"templateCode"`
}

// 日志配置
type LogConfig struct {
	LogPath string `toml:"logPath"`
}

type KafkaConfig struct {
	MessageMode string        `toml:"messageMode"`
	HostPort    string        `toml:"hostPort"`
	LoginTopic  string        `toml:"loginTopic"`
	LogoutTopic string        `toml:"logoutTopic"`
	ChatTopic   string        `toml:"chatTopic"`
	Partition   int           `toml:"partition"`
	Timeout     time.Duration `toml:"timeout"`
}

type StaticSrcConfig struct {
	StaticAvatarPath string `toml:"staticAvatarPath"`
	StaticFilePath   string `toml:"staticFilePath"`
}

type Config struct {
	MainConfig      `toml:"mainConfig"`
	MysqlConfig     `toml:"mysqlConfig"`
	RedisConfig     `toml:"redisConfig"`
	AuthCodeConfig  `toml:"authCodeConfig"`
	LogConfig       `toml:"logConfig"`
	KafkaConfig     `toml:"kafkaConfig"`
	StaticSrcConfig `toml:"staticSrcConfig"`
}

var config *Config

func LoadConfig() error {
	// 本地部署
	// if _, err := toml.DecodeFile("F:\\go\\kama-chat-server\\configs\\config_local.toml", config); err != nil {
	// 	log.Fatal(err.Error())
	// 	return err
	// }
	// Ubuntu22.04云服务器部署
	// if _, err := toml.DecodeFile("/root/project/KamaChat/configs/config_local.toml", config); err != nil {
	// 	log.Fatal(err.Error())
	// 	return err
	// }
	// return nil

	// Linux本地 devserver
	if _, err := toml.DecodeFile("/home/L1nx32b/Documents/GoProj/go-IM/GoChat/configs/config.toml", config); err != nil {
		log.Fatal(err.Error())
		return err
	}
	return nil
}


func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		_ = LoadConfig()
	}
	return config
}
