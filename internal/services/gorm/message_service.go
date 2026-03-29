package gorm

import (
	"GoChatServer/internal/config"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type messageService struct{}

var MessageService = new(messageService)

// GetMessageList 获取用户间聊天记录
func (m *messageService) GetMessageList(userOneId, userTwoId string) (string, []respond.GetMessageListRespond, int) {
	// 先从redis缓存中查看是否有消息缓存
	rspString, err := myredis.GetKeyWithPrefixNilIsErr("message_list_" + userOneId + "_" + userTwoId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zaplog.Info(err.Error())
			zaplog.Info(fmt.Sprintf("%s %s redis lose", userOneId, userTwoId))

			var messageList []model.Message
			// 查询两人之间的数据
			if res := mysql.GormDB.Where("(send_id = ? AND receive_id = ?) OR (receive_id = ? AND send_id = ?)", userOneId, userTwoId, userTwoId, userOneId).Order("created_at ASC").Find(&messageList); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			// 返回给前端
			var rspList []respond.GetMessageListRespond
			for _, message := range messageList {
				rspList = append(rspList, respond.GetMessageListRespond{
					SendId:     message.SendId,
					SendName:   message.SendName,
					SendAvatar: message.SendAvatar,
					ReceiveId:  message.ReceiveId,
					Content:    message.Content,
					Url:        message.Url,
					Type:       message.Type,
					FileType:   message.FileType,
					FileName:   message.FileName,
					FileSize:   message.FileSize,
					CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
				})
				//rspString, err := json.Marshal(rspList)
				//if err != nil {
				//	zlog.Error(err.Error())
				//}
				//if err := myredis.SetKeyEx("message_list_"+userOneId+"_"+userTwoId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				//	zlog.Error(err.Error())
				//}
				return "获取聊天记录成功", rspList, 0
			}
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// 在缓存中有数据
	var rsp []respond.GetMessageListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取聊天记录成功", rsp, 0
}

// GetGroupMessageList 获取群聊消息记录
func (m *messageService) GetGroupMessageList(groupId string) (string, []respond.GetGroupMessageListRespond, int) {
	// 先查看 redis 缓存中是否存有group聊天缓存
	rspString, err := myredis.GetKeyWithPrefixNilIsErr("group_messagelist_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var messageList []model.Message
			if res := mysql.GormDB.Where("receive_id = ?", groupId).Order("created_at ASC").Find(&messageList); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			var rspList []respond.GetGroupMessageListRespond
			for _, message := range messageList {
				rsp := respond.GetGroupMessageListRespond{
					SendId:     message.SendId,
					SendName:   message.SendName,
					SendAvatar: message.SendAvatar,
					ReceiveId:  message.ReceiveId,
					Content:    message.Content,
					Url:        message.Url,
					Type:       message.Type,
					FileType:   message.FileType,
					FileName:   message.FileName,
					CreatedAt:  message.CreatedAt.Format("2006-01-02 15:04:05"),
				}
				rspList = append(rspList, rsp)
			}
			//rspString, err := json.Marshal(rspList)
			//if err != nil {
			//	zlog.Error(err.Error())
			//}
			//if err := myredis.SetKeyEx("group_messagelist_"+groupId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			//	zlog.Error(err.Error())
			//}
			return "获取聊天记录成功", rspList, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	// redis缓存有数据
	var rsp []respond.GetGroupMessageListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取聊天记录成功", rsp, 0
}

// UploadAvatar 上传头像
func (m *messageService) UploadAvatar(c *gin.Context) (string, int) {
	// 从gin的上下文请求中获取前端传来的头像, 但是不能超过FIle_MAX_SIZE
	if err := c.Request.ParseMultipartForm(constants.FILE_MAX_SIZE); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 多文件上传
	mForm := c.Request.MultipartForm
	for key, _ := range mForm.File {
		file, fileHeader, err := c.Request.FormFile(key)
		if err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer file.Close()
		zaplog.Info(fmt.Sprintf("文件名: %s, 文件大小:%d", fileHeader.Filename, fileHeader.Size))
		// 原来Filename应该是213451545.xxx，将Filename修改为avatar_ownerId.xxx
		ext := filepath.Ext(fileHeader.Filename)
		zaplog.Info(ext)
		localFileName := config.GetConfig().StaticAvatarPath + "/" + fileHeader.Filename
		out, err := os.Create(localFileName)
		if err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer out.Close()
		if _, err := io.Copy(out, file); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		zaplog.Info("上传文件成功")
	}
	return "上传成功", 0
}

// UploadFile 上传文件
func (m *messageService) UploadFile(c *gin.Context) (string, int) {
	if err := c.Request.ParseMultipartForm(constants.FILE_MAX_SIZE); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	mForm := c.Request.MultipartForm
	for key, _ := range mForm.File {
		file, fileHeader, err := c.Request.FormFile(key)
		if err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer file.Close()
		zaplog.Info(fmt.Sprintf("文件名: %s, 文件大小: %d", fileHeader.Filename, fileHeader.Size))
		ext := filepath.Ext(fileHeader.Filename)
		zaplog.Info(ext)
		localFileName := config.GetConfig().StaticFilePath + "/" + fileHeader.Filename
		out, err := os.Create(localFileName)
		if err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		defer out.Close()
		if _, err := io.Copy(out, file); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		zaplog.Info("完成文件上传")
	}
	return "上传成功", 0
}
