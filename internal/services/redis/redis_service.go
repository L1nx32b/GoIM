package myredis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"GoChatServer/internal/config"

	"GoChatServer/pkg/zaplog"

	"github.com/go-redis/redis/v8"
)

/*
redis 作为service和dao层之间的缓冲层，提升访问效率
在该项目中，redis的主要用途
	1. 缓存手机6位随机验证码
	2. 缓存5种列表(联系人用户列表，我创建的群聊列表，我加入的群聊列表，用户会话列表，群聊会话列表)
*/

var redisClient *redis.Client
var ctx = context.Background()

func init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.Host
	port := conf.RedisConfig.Port
	password := conf.RedisConfig.Password
	db := conf.Db
	addr := host + ":" + strconv.Itoa(port)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// 设置超过时间的Key
func SetKeyEx(key string, value string, timeout time.Duration) error {
	err := redisClient.Set(ctx, key, value, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

// 获取key 如果不存在， err=nil
func GetKey(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zaplog.Info("该key不存在")
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// 获取key,如果不存在， err!=nil
func GetKeyNilIsErr(key string) (string, error) {
	value, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

// 获取有Perfix前缀的key, 如果不存在, err!= nil
func GetKeyWithPrefixNilIsErr(prefix string) (string, error) {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = redisClient.Keys(ctx, prefix+"*").Result()
		if err != nil {
			return "", err
		}

		if len(keys) == 0 {
			zaplog.Info("没有找到相关前缀key")
			return "", redis.Nil
		}

		if len(keys) == 1 {
			zaplog.Info(fmt.Sprintln("成功找到了相关前缀key", keys))
			return keys[0], nil
		} else {
			zaplog.Error("找到了数量大于1的key, 查找异常")
			return "", errors.New("找到了数量大于1的key, 查找异常")
		}
	}
}

// 获取有suffix后缀的key, 如果不存在，err!=nil
func GetKeyWithSuffixNilIsErr(suffix string) (string, error) {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = redisClient.Keys(ctx, "*"+suffix).Result()
		if err != nil {
			return "", err
		}

		if len(keys) == 0 {
			zaplog.Info("没有找到相关后缀key")
			return "", redis.Nil
		}

		if len(keys) == 1 {
			zaplog.Info(fmt.Sprintln("成功找到了相关后缀key", keys))
			return keys[0], nil
		} else {
			zaplog.Error("找到了数量大于1的key，查找异常")
			return "", errors.New("找到了数量大于1的key，查找异常")
		}
	}
}

// 如果key存在，删除该key
func DelKeyIfExists(key string) error {
	exists, err := redisClient.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 { // 键存在
		delErr := redisClient.Del(ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	// 无论键是否存在，都不返回错误
	return nil
}

func DelKeysWithPattern(pattern string) error {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = redisClient.Keys(ctx, pattern).Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			log.Println("没有找到对应key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
			log.Println("成功删除相关对应key", keys)
		}
	}

	return nil
}

// 如果有该前缀的key则删除
func DelKeysWithPrefix(prefix string) error {
	//var cursor uint64 = 0
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = redisClient.Keys(ctx, prefix+"*").Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			log.Println("没有找到相关前缀key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
			log.Println("成功删除相关前缀key", keys)
		}
	}

	return nil
}

// 如果有该后缀的key，则删除
func DelKeysWithSuffix(suffix string) error {
	//var cursor uint64 = 0
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = redisClient.Keys(ctx, "*"+suffix).Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			log.Println("没有找到相关后缀key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
			log.Println("成功删除相关后缀key", keys)
		}
	}

	return nil
}

// 删除所有的key
func DeleteAllRedisKey() error {
	var cursor uint64 = 0
	for {
		keys, nextCursor, err := redisClient.Scan(ctx, cursor, "*", 0).Result()
		if err != nil {
			return err
		}
		cursor = nextCursor

		if len(keys) > 0 {
			_, err := redisClient.Del(ctx, keys...).Result()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}
