package gorm

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	dao "GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/contact/contact_status_enum"
	"GoChatServer/pkg/enum/contact/contact_type_enum"
	"GoChatServer/pkg/enum/contact_apply/contact_apply_status_enum"
	"GoChatServer/pkg/enum/group_info/group_status_enum"
	"GoChatServer/pkg/enum/user_info/user_status_enum.go"
	"GoChatServer/pkg/util/random"
	"GoChatServer/pkg/zaplog"
	zlog "GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type userContactService struct{}

var UserContactService = new(userContactService)

// GetUserList 获取用户列表
func (u *userContactService) GetUserList(ownerId string) (string, []respond.MyUserListRespond, int) {
	// redis
	rspString, err := myredis.GetKeyNilIsErr("contact_user_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// dao
			var contactList []model.UserContact
			// 没有被删除
			if res := mysql.GormDB.Debug().Order("created_at DESC").Where("user_id = ? AND status != 4", ownerId).Find(&contactList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					message := "目前不存在联系人"
					zaplog.Info(message)
					return message, nil, 0
				} else {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil, -1
				}
			}

			// dto
			var userListRsp []respond.MyUserListRespond
			for _, contact := range contactList {
				// 联系类型是用户
				if contact.ContactType == contact_type_enum.USER {
					// 获取用户信息
					var user model.UserInfo
					if res := mysql.GormDB.First(&user, "uuid = ?", contact.ContactId); res.Error != nil {
						zaplog.Error(res.Error.Error())
						return constants.SYSTEM_ERROR, nil, -1
					}
					userListRsp = append(userListRsp, respond.MyUserListRespond{
						UserId:   user.Uuid,
						UserName: user.Nickname,
						Avatar:   user.Avatar,
					})
				}
			}
			rspString, err := json.Marshal(userListRsp)
			if err != nil {
				zaplog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("contact_user_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zaplog.Error(err.Error())
			}
			return "获取用户列表成功", userListRsp, 0
		} else {
			zaplog.Error(err.Error())
		}
	}
	var rsp []respond.MyUserListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取用户列表成功", rsp, 0
}

// LoadMyJoinedGroup 获取加入的群聊
func (u *userContactService) LoadMyJoinedGroup(ownerId string) (string, []respond.LoadMyJoinedGroupRespond, int) {
	// redis
	rspString, err := myredis.GetKeyNilIsErr("my_joined_group_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var contactList []model.UserContact
			// 没有退群, 也没有被踢出群聊
			if res := mysql.GormDB.Debug().Order("created_at DESC").Where("user_id = ? AND status != 6 AND status != 7", ownerId).Find(&contactList); res.Error != nil {
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					message := "目前不存在加入的群聊"
					zaplog.Info(message)
					return message, nil, 0
				} else {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil, -1
				}
			}
			var groupList []model.GroupInfo
			for _, contact := range contactList {
				if contact.ContactId[0] == 'G' {
					// 获取群聊信息
					var group model.GroupInfo
					if res := mysql.GormDB.First(&group, "uuid = ?", contact.ContactId); res.Error != nil {
						zaplog.Error(res.Error.Error())
						return constants.SYSTEM_ERROR, nil, -1
					}
					// 群没有被删除, 同时群主是自己
					// 群主删除或者admin删除群聊, status = 7，即被踢出群聊, 不用判断群是否被删除, 因为到不了这一步, 在之前已经判断过了
					if group.OwnerId != ownerId {
						groupList = append(groupList, group)
					}
				}
			}
			var groupListRsp []respond.LoadMyJoinedGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, respond.LoadMyJoinedGroupRespond{
					GroupId:   group.Uuid,
					GroupName: group.Name,
					Avatar:    group.Avatar,
				})
			}
			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zaplog.Error(err.Error())
			}
			if err := myredis.SetKeyEx("my_joined_group_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zaplog.Error(err.Error())
			}
			return "获取加入群成功", groupListRsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.LoadMyJoinedGroupRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zlog.Error(err.Error())
	}
	return "获取加入群成功", rsp, 0
}

// GetContactInfo 获取联系人信息
// 调用这个接口前提是该联系人没有处于删除或被删除, 或者该用户还在群聊中
func (u *userContactService) GetContactInfo(contactId string) (string, respond.GetContactInfoRespond, int) {
	// 群聊
	if contactId[0] == 'G' {
		var group model.GroupInfo
		if res := mysql.GormDB.First(&group, "uuid = ?", contactId); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}
		// 没被禁用
		if group.Status != group_status_enum.DISABLE {
			return "获取联系人信息成功", respond.GetContactInfoRespond{
				ContactId:        group.Uuid,
				ContactName:      group.Name,
				ContactAvatar:    group.Avatar,
				ContactAddMode:   group.AddMode,
				ContactMembers:   group.Members,
				ContactMemberCnt: group.MemberCnt,
				ContactOwnerId:   group.OwnerId,
			}, 0
		} else {
			zaplog.Error("该群聊处于禁用状态")
			return "该聊天处于禁用状态", respond.GetContactInfoRespond{}, -2
		}
	} else { // contactId[0] == 'U'
		// 用户
		var user model.UserInfo
		if res := mysql.GormDB.First(&user, "uuid = ?", contactId); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, respond.GetContactInfoRespond{}, -1
		}
		log.Println(user)
		if user.Status != user_status_enum.DISABLE {
			return "获取联系人信息成功", respond.GetContactInfoRespond{
				ContactId:        user.Uuid,
				ContactName:      user.Nickname,
				ContactAvatar:    user.Avatar,
				ContactBirthday:  user.Birthday,
				ContactEmail:     user.Email,
				ContactPhone:     user.Telephone,
				ContactGender:    user.Gender,
				ContactSignature: user.Signature,
			}, 0
		} else {
			zaplog.Info("该用户处于禁止状态")
			return "该用户处于禁止状态", respond.GetContactInfoRespond{}, -2
		}
	}
}

// DeleteContact 删除联系人(User)
func (u *userContactService) DeleteContact(ownerId, contactId string) (string, int) {
	// status 改为删除
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	// 将用户与联系人的状态标记为删除
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contactId = ?", ownerId, contactId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     contact_status_enum.DELETE,
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 将联系人与用户的状态标记为被删除
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contactId = ?", contactId, ownerId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     contact_status_enum.BE_DELETE,
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删除用户与联系人的会话
	if res := mysql.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删除联系人与用户的会话
	if res := mysql.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", contactId, ownerId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 联系人添加的记录得删除, 这样之后再添加就看最新的申请记录, 如果申请记录是拉黑则无法再添加，如果是拒绝可以再添加
	if res := mysql.GormDB.Model(&model.ContactApply{}).Where("send_id = ? AND receive_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if res := mysql.GormDB.Model(&model.ContactApply{}).Where("send_id = ? AND receive_id = ?", contactId, ownerId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 删除用户的contact_user_list redis缓存
	if err := myredis.DelKeysWithPattern("contact_user_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	return "删除联系人成功", 0
}

// ApplyContact 申请添加联系人
func (u *userContactService) ApplyContact(req request.ApplyContactRequest) (string, int) {
	// 联系人是用户
	if req.ContactId[0] == 'U' {
		var userContact model.UserContact
		if res := mysql.GormDB.First(&userContact, "contact_id = ?", req.ContactId); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zlog.Error("用户不存在")
				return "用户不存在", -2
			} else {
				zlog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		var user model.UserInfo
		if res := mysql.GormDB.First(&user, "uuid = ?", userContact.UserId); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zaplog.Error("用户不存在")
				return "用户不存在", -1
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		if user.Status == user_status_enum.DISABLE {
			zaplog.Info("用户已被禁用")
			return "用户已被禁用", -2
		}

		var contactApply model.ContactApply
		// 搜索请求表的记录
		if res := mysql.GormDB.Where("user_id = ? AND contact_id = ?", req.OwnerId, req.ContactId).First(&contactApply); res.Error != nil {
			// 若是没有与联系人的申请数据
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// 创建联系请求
				contactApply = model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.USER,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}
				if res := mysql.GormDB.Create(&contactApply); res.Error != nil {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, -1
				}
				// TODO 直接返回数据避免冗余
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		// 如果存在申请记录, 先查看是否被拉黑
		if contactApply.Status == contact_apply_status_enum.BLACK {
			return "对方已将你拉黑", -2
		}
		// 如果没有被拉黑, 那么将请求更新
		contactApply.LastApplyAt = time.Now()
		contactApply.Status = contact_apply_status_enum.PENDING

		if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	} else if req.ContactId[0] == 'G' { // 若申请的是群聊
		var group model.GroupInfo
		// 查看群聊是否存在
		if res := mysql.GormDB.First(&group, "uuid = ?", req.ContactId); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zaplog.Info("群聊不存在")
				return "群聊不存在", -2
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		// 查看群聊状态
		if group.Status == group_status_enum.DISABLE {
			zaplog.Error("群聊已被禁用")
			return "群聊已被禁用", -2
		}
		// 创建加入群聊申请
		var contactApply model.ContactApply
		// 查看请求表中是否有数据
		if res := mysql.GormDB.Where("user_id = ? AND contact_id = ?", req.OwnerId, req.ContactId); res.Error != nil {
			// 如果没有记录, 则创建申请
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				contactApply = model.ContactApply{
					Uuid:        fmt.Sprintf("A%s", random.GetNowAndLenRandomString(11)),
					UserId:      req.OwnerId,
					ContactId:   req.ContactId,
					ContactType: contact_type_enum.GROUP,
					Status:      contact_apply_status_enum.PENDING,
					Message:     req.Message,
					LastApplyAt: time.Now(),
				}

				if res := mysql.GormDB.Create(&contactApply); res.Error != nil {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, -1
				}
				// TODO 跳过一下逻辑

			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		// 若有记录, 则更新该记录
		contactApply.LastApplyAt = time.Now()
		if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		return "申请成功", 0
	} else { // 格式错误或者乱填
		return "用户/群聊不存在", -2
	}
}

// GetNewContactList 获取新的联系人申请列表
func (u *userContactService) GetNewContactList(ownerId string) (string, []respond.NewContactListRespond, int) {
	// 查询在申请中的申请表数据
	var userContactInfo model.UserContact
	if res := mysql.GormDB.Where("user_id = ?", ownerId).Where("contact_id LIKE ?", "U%").Find(&userContactInfo); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("没有找到信息")
			return "没有找到信息", nil, 0
		} else {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	var contactApplyList []model.ContactApply
	if res := dao.GormDB.Debug().Where("contact_id = ? AND status = ?", userContactInfo.ContactId, contact_apply_status_enum.PENDING).Find(&contactApplyList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zlog.Info("没有在申请的联系人")
			return "没有在申请的联系人", nil, 0
		} else {
			zlog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.NewContactListRespond
	// 所有contact都没被删除, 状态已经在上面判断过了
	for _, contactApply := range contactApplyList {
		var message string
		if contactApply.Message == "" {
			message = "申请理由: 无"
		} else {
			message = "申请理由: " + contactApply.Message
		}

		newContact := respond.NewContactListRespond{
			ContactId: contactApply.Uuid,
			Message:   message,
		}

		var user model.UserInfo
		if res := mysql.GormDB.First(&user, "uuid = ?", contactApply.UserId); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
		newContact.ContactId = user.Uuid
		newContact.ContactName = user.Nickname
		newContact.ContactAvatar = user.Avatar
		rsp = append(rsp, newContact)
	}
	return "获取申请记录成功", rsp, 0
}

// GetAddGroupList 获取新的加群列表
// 前端已经判断调用接口的用户是群主，也只有群主才能调用这个接口
func (u *userContactService) GetAddGroupList(groupId string) (string, []respond.AddGroupListRespond, int) {
	var contactApplyList []model.ContactApply
	// 查看关于自己群的申请消息(status = PeNDING)
	if res := mysql.GormDB.Where("contactId = ? AND status = ?", groupId, contact_apply_status_enum.PENDING).Find(&contactApplyList); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zaplog.Info("没有申请")
			return "没有申请", nil, 0
		} else {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	var rsp []respond.AddGroupListRespond
	for _, contactApply := range contactApplyList {
		var message string
		if contactApply.Message == "" {
			message = "申请理由: 无"
		} else {
			message = "申请理由: " + contactApply.Message
		}

		newContact := respond.AddGroupListRespond{
			ContactId: contactApply.Uuid,
			Message:   message,
		}

		var user model.UserInfo
		if res := mysql.GormDB.First(&user, "uuid = ?", contactApply.UserId); res.Error != nil {
			return constants.SYSTEM_ERROR, nil, -1
		}

		newContact.ContactId = user.Uuid
		newContact.ContactName = user.Nickname
		newContact.ContactAvatar = user.Avatar
		rsp = append(rsp, newContact)
	}
	return "获取申请记录成功", rsp, 0
}

// PassContactApply 通过申请请求
func (u *userContactService) PassContactApply(ownerId string, contactId string) (string, int) {
	// ownerId如果是用户那么是登录用户, 如果是群聊那么就是群聊id
	var contactInfo model.UserContact
	if res := dao.GormDB.Where("user_id = ?", ownerId).Where("contact_id LIKE ?", "U%").First(&contactInfo); res.Error != nil {
		zlog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	var contactApply model.ContactApply
	if res := mysql.GormDB.Where("contact_id = ? AND user_id = ?", contactInfo.ContactId, contactId).First(&contactApply); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 如果是用户之间处理请求
	if ownerId[0] == 'U' {
		var user model.UserInfo
		if res := mysql.GormDB.Where("uuid = ?", contactId).Find(&user); res != nil {
			zaplog.Error(res.Error.Error())
		}

		// 用户被禁用
		if user.Status == user_status_enum.DISABLE {
			zaplog.Error("用户被禁用")
			return "用户已被禁用", -2
		}

		// 更新请求表的申请信息
		contactApply.Status = contact_apply_status_enum.AGREE
		if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 添加用户和联系人关系表映射
		newContact := model.UserContact{
			UserId:      ownerId,
			ContactId:   contactId,
			ContactType: contact_type_enum.USER,     // 用户
			Status:      contact_status_enum.NORMAL, // 正常联系关系
			CreatedAt:   time.Now(),
			UpdateAt:    time.Now(),
		}
		if res := mysql.GormDB.Create(&newContact); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 添加联系人与用户的关系映射
		anotherContact := model.UserContact{
			UserId:      contactId,
			ContactId:   ownerId,
			ContactType: contact_type_enum.USER,
			Status:      contact_status_enum.NORMAL,
			CreatedAt:   time.Now(),
			UpdateAt:    time.Now(),
		}
		if res := mysql.GormDB.Create(&anotherContact); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		return "已添加联系人", 0
	} else { // 通过群聊申请
		var group model.GroupInfo
		// 获取该群信息
		if res := mysql.GormDB.Where("uuid = ?", ownerId).Find(&group); res.Error != nil {
			zaplog.Error(res.Error.Error())
		}

		if group.Status == group_status_enum.DISABLE {
			zaplog.Error("群聊已被禁用")
			return "群聊已被禁用", -2
		}

		// 更新请求信息
		contactApply.Status = contact_apply_status_enum.AGREE
		if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 群聊只需一条UserContact, 因为一条足以表达双方状态
		newContact := model.UserContact{
			UserId:      contactId,
			ContactId:   ownerId,
			ContactType: contact_type_enum.GROUP,
			Status:      contact_status_enum.NORMAL,
			CreatedAt:   time.Now(),
			UpdateAt:    time.Now(),
		}
		if res := mysql.GormDB.Create(&newContact); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 将申请人添加到群聊成员信息中
		var members []string
		// 反序列化出members
		if err := json.Unmarshal(group.Members, &members); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 更新群聊成员信息
		members = append(members, contactId)
		group.MemberCnt = len(members)
		// 序列化
		group.Members, _ = json.Marshal(members)
		if res := mysql.GormDB.Save(&group); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 删除redis缓存，从数据库的数据重新加载
		if err := myredis.DelKeysWithPattern("my_joined_group_list_" + ownerId); err != nil {
			zaplog.Error(err.Error())
		}
		return "已通过加群申请", 0
	}
}

// RefuseContactApply 拒绝联系人申请
func (u *userContactService) RefuseContactApply(ownerId, contactId string) (string, int) {
	// ownerId如果是用户的话就是登录用户，群聊就是群聊id
	var contactApply model.ContactApply
	// 查找申请记录
	if res := mysql.GormDB.Where("contact_id = ? AND user_id = ?", ownerId, contactId).First(&contactApply); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 更新申请状态
	contactApply.Status = contact_apply_status_enum.REFUSE
	if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if ownerId[0] == 'U' {
		return "已拒绝该联系人申请", 0
	} else {
		return "已拒绝该添加群申请", 0
	}
}

// BlackContact 拒绝联系人
func (u *userContactService) BlackContact(ownerId, contactId string) (string, int) {
	// 拉黑
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", ownerId, contactId).Updates(map[string]interface{}{
		"status":    contact_status_enum.BLACK,
		"update_at": time.Now(),
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 被拉黑
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", contactId, ownerId).Updates(map[string]interface{}{
		"status":    contact_status_enum.BE_BLACK,
		"update_at": time.Now(),
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除会话
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	if res := mysql.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", ownerId, contactId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	return "已拉黑该联系人", 0
}

// CancelBlackContact 取消拉黑联系人
func (u *userContactService) CancelBlackContact(ownerId, contactId string) (string, int) {
	// 先判断双方是否有拉黑的状态
	var blackContact model.UserContact
	if res := mysql.GormDB.Where("user_id = ? AND contact_id = ?", ownerId, contactId).First(&blackContact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if blackContact.Status != contact_status_enum.BLACK {
		return "未拉黑该联系人, 无需解除拉黑状态", -2
	}

	var beBlackContact model.UserContact
	if res := mysql.GormDB.Where("user_id = ? AND contact_id = ?", contactId, ownerId).First(&beBlackContact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if beBlackContact.Status != contact_status_enum.BLACK {
		return "该联系人未被拉黑, 无需解除拉黑状态", -2
	}

	// 取消拉黑状态
	blackContact.Status = contact_status_enum.NORMAL
	beBlackContact.Status = contact_status_enum.NORMAL
	if res := mysql.GormDB.Save(&blackContact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if res := mysql.GormDB.Save(&beBlackContact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	return "已解除拉黑联系人", 0
}

// BlackApply 拉黑申请
func (u *userContactService) BlackApply(ownerId, contactId string) (string, int) {
	var contactApply model.ContactApply
	if res := mysql.GormDB.Where("contact_id = ? AND user_id = ?", ownerId, contactId).First(&contactApply); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 更新申请状态
	contactApply.Status = contact_apply_status_enum.BLACK
	if res := mysql.GormDB.Save(&contactApply); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	return "已拉黑该申请", 0
}
