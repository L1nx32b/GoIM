package gorm

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/contact/contact_status_enum"
	"GoChatServer/pkg/enum/contact/contact_type_enum"
	"GoChatServer/pkg/enum/group_info/group_status_enum"
	"GoChatServer/pkg/util/random"
	"GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type groupInfoService struct{}

var GroupInfoService = new(groupInfoService)

// CreateGroup 创建群聊
func (g *groupInfoService) CreateGroup(groupReq request.CreateGroupRequest) (string, int) {
	group := model.GroupInfo{
		Uuid:      fmt.Sprintf("G%s", random.GetNowAndLenRandomString(11)),
		Name:      groupReq.Name,
		Notice:    groupReq.Notice,
		OwnerId:   groupReq.OwnerId,
		MemberCnt: 1,
		AddMode:   groupReq.AddMode,
		Avatar:    groupReq.Avatar,
		Status:    group_status_enum.NORMAL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 序列化成员id
	var members []string
	members = append(members, group.OwnerId)
	var err error
	group.Members, err = json.Marshal(members)
	if err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if res := mysql.GormDB.Create(&group); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 联系表添加用户 - 群聊关系
	contact := model.UserContact{
		UserId:      groupReq.OwnerId,
		ContactId:   group.Uuid,
		ContactType: contact_type_enum.GROUP,
		Status:      contact_status_enum.NORMAL,
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}
	if res := mysql.GormDB.Create(&contact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 添加redis缓存
	if err := myredis.DelKeysWithPattern("contact_mygroup_list_" + groupReq.OwnerId); err != nil {
		zaplog.Error(err.Error())
	}
	return "创建成功", 0
}

// GetAllMembers 获取所有成员信息

// LoadMyGroup 获取我创建的群聊
func (g *groupInfoService) LoadMyGroup(ownerId string) (string, []respond.LoadMyGroupRespond, int) {
	// 先查看redis缓存
	rspString, err := myredis.GetKeyNilIsErr("contact_mygroup_list_" + ownerId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var groupList []model.GroupInfo
			// 查询数据库
			if res := mysql.GormDB.Order("created_at DESC").Where("owner_id = ?", ownerId).Find(&groupList); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			// 返回给前端
			var groupListRsp []respond.LoadMyGroupRespond
			for _, group := range groupList {
				groupListRsp = append(groupListRsp, respond.LoadMyGroupRespond{
					GroupId:   group.Uuid,
					GroupName: group.Name,
					Avatar:    group.Avatar,
				})
			}

			// 序列化
			rspString, err := json.Marshal(groupListRsp)
			if err != nil {
				zaplog.Error(err.Error())
			}

			// 添加redis缓存
			if err := myredis.SetKeyEx("contact_mygroup_list_"+ownerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
				zaplog.Error(err.Error())
			}
			return "获取成功", groupListRsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// redis中有数据
	var groupListRsp []respond.LoadMyGroupRespond
	if err := json.Unmarshal([]byte(rspString), &groupListRsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取成功", groupListRsp, 0
}

// GetGroupInfo 获取群聊详情
func (g *groupInfoService) GetGroupInfo(groupId string) (string, *respond.GetGroupInfoRespond, int) {
	// 先查看redis缓存
	rspString, err := myredis.GetKeyWithPrefixNilIsErr("group_list_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := mysql.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}

			rsp := &respond.GetGroupInfoRespond{
				Uuid:      group.Uuid,
				Name:      group.Name,
				Notice:    group.Notice,
				Avatar:    group.Avatar,
				MemberCnt: group.MemberCnt,
				OwnerId:   group.OwnerId,
				AddMode:   group.AddMode,
				Status:    group.Status,
			}
			if group.DeletedAt.Valid {
				rsp.IsDeleted = true
			} else {
				rsp.IsDeleted = false
			}
			//rspString, err := json.Marshal(rsp)
			//if err != nil {
			//	zaplog.Error(err.Error())
			//}
			//if err := myredis.SetKeyEx("group_info_"+groupId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			//	zaplog.Error(err.Error())
			//}
			return "获取成功", rsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	// redis缓存中有数据
	var rsp *respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取成功", rsp, 0
}

// GetGroupInfoList 获取群聊列表 - 管理员
// 管理员少，而且如果用户更改了，那么管理员会一直频繁删除redis，更新redis，比较麻烦，所以管理员暂时不使用redis缓存
func (g *groupInfoService) GetGroupInfoList() (string, []respond.GetGroupInfoRespond, int) {
	var groupList []model.GroupInfo
	if res := mysql.GormDB.Unscoped().Find(&groupList); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 传给前端
	var rsp []respond.GetGroupInfoRespond
	for _, group := range groupList {
		rp := respond.GetGroupInfoRespond{
			Uuid:    group.Uuid,
			Name:    group.Name,
			OwnerId: group.OwnerId,
			Status:  group.Status,
		}
		if group.DeletedAt.Valid {
			rp.IsDeleted = true
		} else {
			rp.IsDeleted = false
		}
		rsp = append(rsp, rp)
	}
	return "获取成功", rsp, 0
}

// LeaveGroup 退群
func (g *groupInfoService) LeaveGroup(userId, groupId string) (string, int) {
	// 从群聊中移除该用户
	var group model.GroupInfo
	if res := mysql.GormDB.First(&group, "uuid = ?", groupId); res != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 反序列化, 获取当前群中所有成员id集合
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for i, member := range members {
		if member == userId {
			members = append(members[:i], members[i+1:]...)
			break
		}
	}

	// 将删除用户的成员集合保存回数据库
	if data, err := json.Marshal(members); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	} else {
		group.Members = data
	}
	group.MemberCnt -= 1
	if res := mysql.GormDB.Save(&group); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除用户关于群聊的会话
	var deletedAt gorm.DeletedAt
	deletedAt.Valid = true
	deletedAt.Time = time.Now()
	if res := mysql.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", userId, groupId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除用户-群聊的关系表
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", userId, groupId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"status":     contact_status_enum.QUIT_GROUP,
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 删除申请记录, 后面还可以申请加入
	if res := mysql.GormDB.Model(&model.ContactApply{}).Where("contact_id = ? AND user_id = ?", groupId, userId).Update("deleted_at", deletedAt); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 更新redis缓存
	//if err := myredis.DelKeysWithPattern("group_info_" + groupId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	//if err := myredis.DelKeysWithPattern("groupmember_list_" + groupId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	if err := myredis.DelKeysWithPattern("group_session_list_" + userId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("my_joined_group_list_" + userId); err != nil {
		zaplog.Error(err.Error())
	}
	//if err := myredis.DelKeysWithPattern("session_" + userId + "_" + groupId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	return "退群成功", 0
}

// DismissGroup 解散群聊
func (g *groupInfoService) DismissGroup(ownerId, groupId string) (string, int) {
	var deletedAt gorm.DeletedAt
	deletedAt.Valid = true
	deletedAt.Time = time.Now()
	if res := mysql.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", groupId).Updates(map[string]interface{}{
		"deleted_at": deletedAt,
		"update_at":  deletedAt.Time,
	}); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	// 将会话删除
	var sessionList []model.Session
	if res := mysql.GormDB.Model(&model.Session{}).Where("receive_id = ?", groupId).Find(&sessionList); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, session := range sessionList {
		if res := mysql.GormDB.Model(&session).Updates(map[string]interface{}{
			"deleted_at": deletedAt,
		}); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}

	// 将联系列表删除
	var userContactList []model.UserContact
	if res := mysql.GormDB.Model(&model.UserContact{}).Where("contact_id = ?", groupId).Find(&userContactList); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}

	for _, userContact := range userContactList {
		if res := mysql.GormDB.Model(&userContact).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}

	// 将申请记录删除
	var contactApplys []model.ContactApply
	if res := mysql.GormDB.Model(&contactApplys).Where("contact_id = ?", groupId).Find(&contactApplys); res.Error != nil {
		if res.Error != gorm.ErrRecordNotFound {
			zaplog.Info(res.Error.Error())
			return "无响应的申请记录需要删除", 0
		}
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, contactApply := range contactApplys {
		if res := mysql.GormDB.Model(&contactApply).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}

	// 删除redis 缓存
	if err := myredis.DelKeysWithPattern("contact_mygroup_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPrefix("my_joined_group_list"); err != nil {
		zaplog.Error(err.Error())
	}
	return "解散群聊成功", 0
}

// DeleteGroups 删除列表中群聊 - 管理员
func (g *groupInfoService) DeleteGroups(uuidList []string) (string, int) {
	for _, uuid := range uuidList {
		var deletedAt gorm.DeletedAt
		deletedAt.Time = time.Now()
		deletedAt.Valid = true
		if res := mysql.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", uuid).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 删除会话
		var sessionList []model.Session
		if res := mysql.GormDB.Model(&model.Session{}).Where("receive_id = ?", uuid).Find(&sessionList); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		for _, session := range sessionList {
			if res := mysql.GormDB.Model(&session).Update("deleted_at", deletedAt); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		// 删除联系人
		var userContactList []model.UserContact
		if res := mysql.GormDB.Model(&model.UserContact{}).Where("contact_id = ?", uuid).Find(&userContactList); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		for _, userContact := range userContactList {
			if res := mysql.GormDB.Model(&userContact).Update("deleted_at", deletedAt); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		var contactApplys []model.ContactApply
		if res := mysql.GormDB.Model(&contactApplys).Where("contact_id = ?", uuid).Find(&contactApplys); res.Error != nil {
			if res.Error != gorm.ErrRecordNotFound {
				zaplog.Info(res.Error.Error())
				return "无响应的申请记录需要删除", 0
			}
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		for _, contactApply := range contactApplys {
			if res := mysql.GormDB.Model(&contactApply).Update("deleted_at", deletedAt); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
	}
	//for _, uuid := range uuidList {
	//	if err := myredis.DelKeysWithPattern("group_info_" + uuid); err != nil {
	//		zaplog.Error(err.Error())
	//	}
	//	if err := myredis.DelKeysWithPattern("groupmember_list_" + uuid); err != nil {
	//		zaplog.Error(err.Error())
	//	}
	//}
	if err := myredis.DelKeysWithPrefix("contact_mygroup_list"); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPrefix("group_session_list"); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPrefix("group_session_list"); err != nil {
		zaplog.Error(err.Error())
	}
	return "解散/删除群聊成功", 0
}

// CheckGroupAddMode 检查群聊加群方式
func (g *groupInfoService) CheckGroupAddMode(groupId string) (string, int8, int) {
	rspString, err := myredis.GetKeyNilIsErr("group_info_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := mysql.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1, -1
			}
			return "加群方式获取成功", group.AddMode, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, -1, -1
		}
	}
	var rsp respond.GetGroupInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "加群方式获取成功", rsp.AddMode, 0
}

// EnterGroupDirectly 直接进群
// ownerId 是群聊id
func (g *groupInfoService) EnterGroupDirectly(ownerId, contactId string) (string, int) {
	var group model.GroupInfo
	if res := mysql.GormDB.First(&group, "uuid = ?", ownerId); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	members = append(members, contactId)
	if data, err := json.Marshal(members); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	} else {
		group.Members = data
	}
	group.MemberCnt += 1
	if res := mysql.GormDB.Save(&group); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	newContact := model.UserContact{
		UserId:      contactId,
		ContactId:   ownerId,
		ContactType: contact_type_enum.GROUP,    // 用户
		Status:      contact_status_enum.NORMAL, // 正常
		CreatedAt:   time.Now(),
		UpdateAt:    time.Now(),
	}
	if res := mysql.GormDB.Create(&newContact); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	//if err := myredis.DelKeysWithPattern("group_info_" + contactId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	//if err := myredis.DelKeysWithPattern("groupmember_list_" + contactId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	if err := myredis.DelKeysWithPattern("group_session_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPattern("my_joined_group_list_" + ownerId); err != nil {
		zaplog.Error(err.Error())
	}
	//if err := myredis.DelKeysWithPattern("session_" + ownerId + "_" + contactId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	return "进群成功", 0
}

// SetGroupsStatus 设置群聊是否启用
func (g *groupInfoService) SetGroupsStatus(uuidList []string, status int8) (string, int) {
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	for _, uuid := range uuidList {
		if res := mysql.GormDB.Model(&model.GroupInfo{}).Where("uuid = ?", uuid).Update("status", status); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		if status == group_status_enum.DISABLE {
			var sessionList []model.Session
			if res := mysql.GormDB.Model(&sessionList).Where("receive_id = ?", uuid).Find(&sessionList); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
			for _, session := range sessionList {
				if res := mysql.GormDB.Model(&session).Update("deleted_at", deletedAt); res.Error != nil {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, -1
				}
			}
		}
	}
	//for _, uuid := range uuidList {
	//	if err := myredis.DelKeysWithPattern("group_info_" + uuid); err != nil {
	//		zaplog.Error(err.Error())
	//	}
	//}
	return "设置成功", 0
}

// UpdateGroupInfo 更新群聊消息
func (g *groupInfoService) UpdateGroupInfo(req request.UpdateGroupInfoRequest) (string, int) {
	var group model.GroupInfo
	if res := mysql.GormDB.First(&group, "uuid = ?", req.Uuid); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.AddMode != -1 {
		group.AddMode = req.AddMode
	}
	if req.Notice != "" {
		group.Notice = req.Notice
	}
	if req.Avatar != "" {
		group.Avatar = req.Avatar
	}
	if res := mysql.GormDB.Save(&group); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	// 修改会话
	var sessionList []model.Session
	if res := mysql.GormDB.Where("receive_id = ?", req.Uuid).Find(&sessionList); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, session := range sessionList {
		session.ReceiveName = group.Name
		session.Avatar = group.Avatar
		log.Println(session)
		if res := mysql.GormDB.Save(&session); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}

	//if err := myredis.DelKeysWithPattern("group_info_" + req.Uuid); err != nil {
	//	zaplog.Error(err.Error())
	//}
	//if err := myredis.SetKeyEx("contact_mygroup_list_"+ req.OwnerId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
	//	zaplog.Error(err.Error())
	//}
	return "更新成功", 0
}

// GetGroupMemberList 获取群聊成员列表
func (g *groupInfoService) GetGroupMemberList(groupId string) (string, []respond.GetGroupMemberListRespond, int) {
	rspString, err := myredis.GetKeyNilIsErr("group_memberlist_" + groupId)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			var group model.GroupInfo
			if res := mysql.GormDB.First(&group, "uuid = ?", groupId); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			var members []string
			if err := json.Unmarshal(group.Members, &members); err != nil {
				zaplog.Error(err.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			var rspList []respond.GetGroupMemberListRespond
			for _, member := range members {
				var user model.UserInfo
				if res := mysql.GormDB.First(&user, "uuid = ?", member); res.Error != nil {
					zaplog.Error(res.Error.Error())
					return constants.SYSTEM_ERROR, nil, -1
				}
				rspList = append(rspList, respond.GetGroupMemberListRespond{
					UserId:   user.Uuid,
					Nickname: user.Nickname,
					Avatar:   user.Avatar,
				})
			}
			//rspString, err := json.Marshal(rspList)
			//if err != nil {
			//	zaplog.Error(err.Error())
			//}
			//if err := myredis.SetKeyEx("group_memberlist_"+groupId, string(rspString), time.Minute*constants.REDIS_TIMEOUT); err != nil {
			//	zaplog.Error(err.Error())
			//}
			return "获取群聊成员列表成功", rspList, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp []respond.GetGroupMemberListRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取群聊成员列表成功", rsp, 0
}

// RemoveGroupMembers 移除群聊成员
func (g *groupInfoService) RemoveGroupMembers(req request.RemoveGroupMembersRequest) (string, int) {
	// 查找
	var group model.GroupInfo
	if res := mysql.GormDB.First(&group, "uuid = ?", req.GroupId); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	var members []string
	if err := json.Unmarshal(group.Members, &members); err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	var deletedAt gorm.DeletedAt
	deletedAt.Time = time.Now()
	deletedAt.Valid = true
	log.Println(req.UuidList, req.OwnerId)
	for _, uuid := range req.UuidList {
		if req.OwnerId == uuid {
			return "不能移除群主", -2
		}
		// 从members中找到uuid，移除
		for i, member := range members {
			if member == uuid {
				members = append(members[:i], members[i+1:]...)
			}
		}
		group.MemberCnt -= 1
		// 删除会话
		if res := mysql.GormDB.Model(&model.Session{}).Where("send_id = ? AND receive_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 删除联系人
		if res := mysql.GormDB.Model(&model.UserContact{}).Where("user_id = ? AND contact_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		// 删除申请记录
		if res := mysql.GormDB.Model(&model.ContactApply{}).Where("user_id = ? AND contact_id = ?", uuid, req.GroupId).Update("deleted_at", deletedAt); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}
	group.Members, _ = json.Marshal(members)
	if res := mysql.GormDB.Save(&group); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	//if err := myredis.DelKeysWithPattern("group_info_" + req.GroupId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	//if err := myredis.DelKeysWithPattern("groupmember_list_" + req.GroupId); err != nil {
	//	zaplog.Error(err.Error())
	//}
	if err := myredis.DelKeysWithPrefix("group_session_list"); err != nil {
		zaplog.Error(err.Error())
	}
	if err := myredis.DelKeysWithPrefix("my_joined_group_list"); err != nil {
		zaplog.Error(err.Error())
	}
	return "移除群聊成员成功", 0
}
