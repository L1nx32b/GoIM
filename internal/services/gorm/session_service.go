package gorm

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/contact/contact_status_enum"
	"GoChatServer/pkg/enum/group_info/group_status_enum"
	"GoChatServer/pkg/enum/user_info/user_status_enum.go"
	"GoChatServer/pkg/util/random"
	"GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type sessionService struct{}

var SessionService = new(sessionService)

// CreateSession 创建会话
func (s *sessionService) CreateSession(req request.CreateSessionRequest) (string, string, int) {
	var user model.UserInfo
	// 先查看申请会话的人的用户是否存在
	if res := mysql.GormDB.Where("uuid = ?", req.SendId).First(&user); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	// 创建基础会话信息
	var session model.Session
	session.Uuid = fmt.Sprintf("S%s", random.GetNowAndLenRandomString(11))
	session.SendId = req.SendId
	session.ReceiveId = req.ReceiveId
	session.CreatedAt = time.Now()

	if req.ReceiveId[0] == 'U' { // 用户会话
		var receiveUser model.UserInfo
		// 先检查会话人是否存在
		if res := mysql.GormDB.Where("uuid = ?", req.ReceiveId).First(&receiveUser); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, "", -1
		}

		// 该用户被禁用了
		if receiveUser.Status == user_status_enum.DISABLE {
			zaplog.Error("该用户已被禁用")
			return "该用户被禁用", "", -2
		} else {
			session.ReceiveName = receiveUser.Nickname
			session.Avatar = receiveUser.Avatar
		}
	} else { // 群聊会话
		var receiveGroup model.GroupInfo
		if res := mysql.GormDB.Where("uuid = ?", req.ReceiveId).First(&receiveGroup); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, "", -1
		}

		if receiveGroup.Status == group_status_enum.DISABLE {
			zaplog.Error("该群聊被禁用了")
			return "该群聊被禁用", "", -2
		} else {
			session.ReceiveName = receiveGroup.Name
			session.Avatar = receiveGroup.Avatar
		}
	}

	if res := mysql.GormDB.Create(&session); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, "", -1
	}

	// 删除sendId的缓存，重新加载
	if err := myredis.DelKeysWithPattern("group_session_list_" + req.SendId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("session_list_" + req.ReceiveId); err != nil {
		zaplog.Error(err.Error())
	}

	return "会话创建成功", session.Uuid, 0
}

// CheckOpenSessionAllowed 检查是否允许发起会话
func (u *sessionService) CheckOpenSessionAllowed(sendId, receiveId string) (string, bool, int) {
	var contact model.UserContact
	// 查看关系表中是否存在数据
	if res := mysql.GormDB.Where("user_id = ? AND contact_id = ?", sendId, receiveId).First(&contact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, false, -1
	}

	// 查看是否被拉黑
	if contact.Status == contact_status_enum.BE_BLACK {
		return "已被对方拉黑, 无法发起会话", false, -2
	} else if contact.Status == contact_status_enum.BLACK {
		return "已把对方拉黑, 先解除拉黑状态才能发起会话", false, -2
	}

	if receiveId[0] == 'U' { // 会话对象是用户
		var user model.UserInfo
		// 查看对方是否存在
		if res := mysql.GormDB.Where("uuid = ?", receiveId).First(&user); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, false, -1
		}
		// 用户被禁用
		if user.Status == user_status_enum.DISABLE {
			zaplog.Info("对方被禁用, 无法发起会话")
			return "对方已被禁用, 无法发起会话", false, -2
		}
	} else { // 会话对象是群聊
		var group model.GroupInfo
		// 查看群聊是否存在
		if res := mysql.GormDB.Where("uuid = ?", receiveId).First(&group); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, false, -1
		}
		// 群聊被禁用
		if group.Status == group_status_enum.DISABLE {
			zaplog.Info("群聊已被禁用, 无法发起会话")
			return "群聊已被禁用, 无法发起会话", false, -2
		}
	}
	return "可以发起会话", true, 0
}

// OpenSession 打开会话
func (s *sessionService) OpenSession(req request.OpenSessionRequest) (string, string, int) {
	// 先查看redis缓存中是否存在数据
	rspString, err := myredis.GetKeyWithPrefixNilIsErr("session_" + req.SendId + "_" + req.ReceiveId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var session model.Session
			if res := mysql.GormDB.Where("send_id = ? AND receive_id", req.SendId, req.ReceiveId).First(&session); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zaplog.Info("没有找到会话, 创建新会话")
					createReq := request.CreateSessionRequest{
						SendId:    req.SendId,
						ReceiveId: req.ReceiveId,
					}
					return s.CreateSession(createReq)
				}
			}
			//rspString, err := json.Marshal(session)
			//if err != nil {
			//	zlog.Error(err.Error())
			//}
			//if err := myredis.SetKeyEx("session_"+req.SendId+"_"+req.ReceiveId+"_"+session.Uuid, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			//	zlog.Error(err.Error())
			//}
			return "会话创建成功", session.Uuid, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, "", -1
		}
	}

	var session model.Session
	if err := json.Unmarshal([]byte(rspString), &session); err != nil {
		zaplog.Error(err.Error())
	}
	return "会话创建成功", session.Uuid, 0
}

// GetUserSessionList 获取用户会话列表
func (s *sessionService) GetUserSessionList(ownerId string) (string, []respond.UserSessionListRespond, int) {
	// 先从redis 缓存中找
	rspString, err := myredis.GetKeyNilIsErr("session_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var sessionList []model.Session
			// 数据库查询会话列表
			if res := mysql.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zaplog.Info("未创建用户会话")
					return "未创建用户会话", nil, 0
				} else {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var sessionListRsp []respond.UserSessionListRespond
			// 遍历所有会话, 并只收集用户类会话
			for i := 0; i < len(sessionList); i++ {
				if sessionList[i].ReceiveId[0] == 'U' {
					sessionListRsp = append(sessionListRsp, respond.UserSessionListRespond{
						SessionId: sessionList[i].Uuid,
						Avatar:    sessionList[i].Avatar,
						UserId:    sessionList[i].ReceiveId,
						Username:  sessionList[i].ReceiveName,
					})
				}
			}
			rspString, err := json.Marshal(sessionListRsp)
			if err != nil {
				zaplog.Error(err.Error())
			}
			// 添加redis缓存
			if err := myredis.SetKeyEx("session_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zaplog.Error(err.Error())
			}
			return "获取成功", sessionListRsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	// 若存在redis缓存
	var rsp []respond.UserSessionListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取成功", rsp, 0
}

// GetGroupSessionList 获取群聊话列表
func (s *sessionService) GetGroupSessionList(ownerId string) (string, []respond.GroupSessionListRespond, int) {
	rspString, err := myredis.GetKeyNilIsErr("group_session_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var sessionList []model.Session
			if res := mysql.GormDB.Order("created_at DESC").Where("send_id = ?", ownerId).Find(&sessionList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					zaplog.Info("未创建群聊会话")
					return "未创建群聊会话", nil, 0
				} else {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var sessionListRsp []respond.GroupSessionListRespond
			for i := 0; i < len(sessionList); i++ {
				if sessionList[i].ReceiveId[0] == 'G' {
					sessionListRsp = append(sessionListRsp, respond.GroupSessionListRespond{
						SessionId: sessionList[i].Uuid,
						Avatar:    sessionList[i].Avatar,
						GroupId:   sessionList[i].ReceiveId,
						GroupName: sessionList[i].ReceiveName,
					})
				}
			}
			rspString, err := json.Marshal(sessionListRsp)
			if err != nil {
				zaplog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("group_session_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zaplog.Error(err.Error())
			}
			return "获取成功", sessionListRsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.GroupSessionListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取成功", rsp, 0
}

// DeleteSession 删除会话
func (s *sessionService) DeleteSession(ownerId, sessionId string) (string, int) {
	var session model.Session
	// 查找会话数据
	if res := mysql.GormDB.Where("uuid = ?", sessionId).Find(&session); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 更新会话删除信息
	session.DeletedAt.Valid = true
	session.DeletedAt.Time = time.Now()
	if res := mysql.GormDB.Save(&session); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除ownerId的会话缓存
	if err := myredis.DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("session_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	return "删除成功", 0
}
