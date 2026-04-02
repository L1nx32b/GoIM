package gorm

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/dto/respond"
	"GoChatServer/internal/model"
	"GoChatServer/internal/mysql"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/internal/services/sms"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/enum/user_info/user_status_enum.go"
	"GoChatServer/pkg/util/random"
	"GoChatServer/pkg/zaplog"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"

	"gorm.io/gorm"
)

type userInfoService struct{}

var UserInfoService = new(userInfoService)

// 校验
// 校验电话是否有效
func (u *userInfoService) checkTelephoneValid(telephone string) bool {
	pattern := `^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`
	match, err := regexp.MatchString(pattern, telephone)
	if err != nil {
		zaplog.Error(err.Error())
	}
	return match
}

// checkEmailValid 校验邮箱是否有效
func (u *userInfoService) checkEmailValid(email string) bool {
	pattern := `^[^\s@]+@[^\s@]+\.[^\s@]+$`
	match, err := regexp.MatchString(pattern, email)
	if err != nil {
		zaplog.Error(err.Error())
	}
	return match
}

// 检验用户是否为管理员
func (u *userInfoService) checkUserIsAdminOrNot(user model.UserInfo) int8 {
	return user.IsAdmin
}

// Login登录(service+mysql)
func (u *userInfoService) Login(loginReq request.LoginRequest) (string, *respond.LoginRespond, int) {
	// 获取前端传来的登录密码
	password := loginReq.Password
	// 接收查询数据库后的数据存储结构体
	var user model.UserInfo
	// 查询对应登录用户电话对应数据库数据
	res := mysql.GormDB.First(&user, "telephone = ?", loginReq.Telephone)

	// 如果查询错误
	if res.Error != nil {
		// 查不到数据
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			message := "用户不存在, 请注册"
			zaplog.Error(message)
			return message, nil, -2
		}
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	// 密码错误
	if user.Password != password {
		message := "密码不正确, 请重试!"
		zaplog.Error(message)
		return message, nil, -2
	}

	// 登录成功后，将用户信息包装到LoginRespond上
	loginRsp := &respond.LoginRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Birthday:  user.Birthday,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	// 将登录信息传给前端作为前端的session
	return "登录成功", loginRsp, 0
}

// SendSmsCode 验证码
func (u *userInfoService) SendSmsCode(telephone string) (string, int) {
	return sms.VerificationCode(telephone)
}

// SmsLogin 验证码登录
func (u *userInfoService) SmsLogin(req request.SmsLoginRequest) (string, *respond.LoginRespond, int) {
	var user model.UserInfo
	res := mysql.GormDB.First(&user, "telephone = ?", req.Telephone)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			message := "用户不存在, 请注册"
			zaplog.Error(message)
			return message, nil, -2
		}
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	key := "auth_code_" + req.Telephone
	code, err := myredis.GetKey(key)
	if err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}

	if code != req.SmsCode {
		message := "验证码不正确, 请重新输入"
		zaplog.Info(message)
		return message, nil, -2
	} else {
		if err := myredis.DelKeyIfExists(key); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	loginRsp := &respond.LoginRespond{
		Uuid:      user.Uuid,
		Telephone: user.Telephone,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Gender:    user.Gender,
		Birthday:  user.Birthday,
		Signature: user.Signature,
		IsAdmin:   user.IsAdmin,
		Status:    user.Status,
	}
	year, month, day := user.CreatedAt.Date()
	loginRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "登录成功", loginRsp, 0
}

// TODO: 邮箱验证码登录
func (u *userInfoService) EmailLogin(Email string) {

}

// TODO: 检查邮箱是否存在

// 检查手机号是否存在
func (u *userInfoService) checkTelephoneExist(telephone string) (string, int) {
	var user model.UserInfo
	// gorm默认软删除
	if res := mysql.GormDB.Where("telephone = ?", telephone).First(&user); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			zaplog.Info("该电话未被注册")
			return "", 0
		}
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	zaplog.Info("该电话已经存在")
	return "该电话已经存在", -2
}

// register
func (u *userInfoService) Register(registerReq request.RegisterRequest) (string, *respond.RegisterRespond, int) {
	key := "auth_code_" + registerReq.Telephone
	code, err := myredis.GetKey(key)
	if err != nil {
		zaplog.Error(err.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	if code != registerReq.SmsCode {
		message := "验证码不正确, 请重试!"
		zaplog.Info(message)
		return message, nil, -2
	} else {
		if err := myredis.DelKeyIfExists(key); err != nil {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}

	// 不用校验手机号，前端校验
	// 判断电话是否已经被注册过了
	message, ret := u.checkTelephoneExist(registerReq.Telephone)
	if ret != 0 {
		return message, nil, ret
	}
	var newUser model.UserInfo
	newUser.Uuid = "U" + random.GetNowAndLenRandomString(11)
	newUser.Telephone = registerReq.Telephone
	newUser.Password = registerReq.Password
	newUser.Nickname = registerReq.Nickname
	newUser.Avatar = "https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png"
	newUser.CreatedAt = time.Now()
	newUser.IsAdmin = u.checkUserIsAdminOrNot(newUser)
	newUser.Status = user_status_enum.NORMAL
	// 手机号验证，最后一步才调用api，省钱hhh
	//err := sms.VerificationCode(registerReq.Telephone)
	//if err != nil {
	//	zaplog.Error(err.Error())
	//	return "", err
	//}

	res := mysql.GormDB.Create(&newUser)
	if res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	// 注册成功，chat client建立
	//if err := chat.NewClientInit(c, newUser.Uuid); err != nil {
	//	return "", err
	//}
	registerRsp := &respond.RegisterRespond{
		Uuid:      newUser.Uuid,
		Telephone: newUser.Telephone,
		Nickname:  newUser.Nickname,
		Email:     newUser.Email,
		Avatar:    newUser.Avatar,
		Gender:    newUser.Gender,
		Birthday:  newUser.Birthday,
		Signature: newUser.Signature,
		IsAdmin:   newUser.IsAdmin,
		Status:    newUser.Status,
	}
	year, month, day := newUser.CreatedAt.Date()
	registerRsp.CreatedAt = fmt.Sprintf("%d.%d.%d", year, month, day)

	return "注册成功", registerRsp, 0
}

// UpdateUserInfo 修改用户信息
// 某用户修改了信息，可能会影响contact_user_list，不需要删除redis的contact_user_list，timeout之后会自己更新
// 但是需要更新redis的user_info，因为可能影响用户搜索
func (u *userInfoService) UpdateUserInfo(updateReq request.UpdateUserInfoRequest) (string, int) {
	var user model.UserInfo
	if res := mysql.GormDB.First(&user, "uuid = ?", updateReq.Uuid); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	if updateReq.Email != "" {
		user.Email = updateReq.Email
	}
	if updateReq.Nickname != "" {
		user.Nickname = updateReq.Nickname
	}
	if updateReq.Birthday != "" {
		user.Birthday = updateReq.Birthday
	}
	if updateReq.Signature != "" {
		user.Signature = updateReq.Signature
	}
	if updateReq.Avatar != "" {
		user.Avatar = updateReq.Avatar
	}
	// 更新用户信息
	if res := mysql.GormDB.Save(&user); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	return "修改用户信息成功", 0
}

// GetUserInfoList 获取用户列表除了ownerId之外 - 管理员
// 管理员少, 如果用户更改，那么管理员会一直频繁删除redis，更新redis，比较繁琐，所以管理员暂不用redis缓存
func (u *userInfoService) GetUserInfoList(ownerId string) (string, []respond.GetUserListRespond, int) {
	// redis中没有数据，从数据库中取
	var users []model.UserInfo
	// 获取所有用户
	if res := mysql.GormDB.Unscoped().Where("uuid != ?", ownerId).Find(&users); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, nil, -1
	}
	var rsp []respond.GetUserListRespond
	for _, user := range users {
		rp := respond.GetUserListRespond{
			Uuid:      user.Uuid,
			Telephone: user.Telephone,
			Nickname:  user.Nickname,
			Status:    user.Status,
			IsAdmin:   user.IsAdmin,
		}
		if user.DeletedAt.Valid {
			rp.IsDeleted = true
		} else {
			rp.IsDeleted = false
		}
		rsp = append(rsp, rp)
	}
	return "获取用户列表成功", rsp, 0
}

// AbleUsers 启用用户
// 用户是否启用禁用需要实时更新contact_user_list状态，所以redis的contact_user_list需要删除
func (u *userInfoService) AbleUsers(uuidlist []string) (string, int) {
	var users []model.UserInfo
	if res := mysql.GormDB.Model(model.UserInfo{}).Where("uuid in (?)", uuidlist).Find(&users); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, user := range users {
		user.Status = user_status_enum.NORMAL
		if res := mysql.GormDB.Save(&user); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}
	// 删除redis缓存中所有"contact_user_list"开头的key
	//if err := myredis.DelKeysWithPrefix("contact_user_list"); err != nil {
	//	zaplog.Error(err.Error())
	//}

	return "启用用户成功", 0
}

// DisableUsers 禁用用户
// 用户是否启用禁用需要实时更新contact_user_list状态，所以redis的contact_user_list需要删除
func (u *userInfoService) DisableUsers(uuidlist []string) (string, int) {
	var users []model.UserInfo
	if res := mysql.GormDB.Model(model.UserInfo{}).Where("uuid in (?)", uuidlist).Find(&users); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, user := range users {
		user.Status = user_status_enum.DISABLE
		if res := mysql.GormDB.Save(&user); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		var sessionList []model.Session
		if res := mysql.GormDB.Where("send_id = ? OR receive_id = ?", user.Uuid, user.Uuid).Find(&sessionList); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
		for _, session := range sessionList {
			var deletedAt gorm.DeletedAt
			deletedAt.Time = time.Now()
			deletedAt.Valid = true
			session.DeletedAt = deletedAt
			if res := mysql.GormDB.Save(&session); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
	}
	// 删除redis缓存中所有'contact_user_list'开头的key
	// if err := myredis.DelKeysWithPrefix("contact_user_list"); err != nil {
	// 	zaplog.Error(err.Error())
	// }

	return "禁用用户成功", 0
}

// DeleteUsers 删除用户
// 用户是否启用禁用需要实时更新contact_user_list状态, 所以redis的contact_user_list需要删除
func (u *userInfoService) DeleteUsers(uuidList []string) (string, int) {
	var users []model.UserInfo
	if res := mysql.GormDB.Model(model.UserInfo{}).Where("uuid in (?)", uuidList).Find(&users); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, user := range users {
		user.DeletedAt.Valid = true
		user.DeletedAt.Time = time.Now()
		if res := mysql.GormDB.Save(&user); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}

		// 删除会话
		var sessionList []model.Session
		if res := mysql.GormDB.Where("send_id = ? OR receive_id = ?", user.Uuid, user.Uuid).Find(&sessionList); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zaplog.Info(res.Error.Error())
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		for _, session := range sessionList {
			var deletedAt gorm.DeletedAt
			deletedAt.Time = time.Now()
			deletedAt.Valid = true
			session.DeletedAt = deletedAt
			if res := mysql.GormDB.Save(&session); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		// 删除联系人
		var contactList []model.UserContact
		if res := mysql.GormDB.Where("user_id = ? OR contact_id = ?", user.Uuid, user.Uuid).Find(&contactList); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zaplog.Info(res.Error.Error())
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		for _, contact := range contactList {
			var deletedAt gorm.DeletedAt
			deletedAt.Time = time.Now()
			deletedAt.Valid = true
			contact.DeletedAt = deletedAt
			if res := mysql.GormDB.Save(&contact); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}

		// 删除申请记录
		var applyList []model.ContactApply
		if res := mysql.GormDB.Where("user_id = ? or contact_id = ?", user.Uuid, user.Uuid).Find(&applyList); res.Error != nil {
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				zaplog.Info(res.Error.Error())
			} else {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
		for _, apply := range applyList {
			var deletedAt gorm.DeletedAt
			deletedAt.Time = time.Now()
			deletedAt.Valid = true
			apply.DeletedAt = deletedAt
			if res := mysql.GormDB.Save(&apply); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, -1
			}
		}
	}
	// 删除所有"contact_user_list"开头的key
	//if err := myredis.DelKeysWithPrefix("contact_user_list"); err != nil {
	//	zaplog.Error(err.Error())
	//}
	return "删除用户成功", 0
}

// GetUserInfo 获取用户信息
func (u *userInfoService) GetUserInfo(uuid string) (string, *respond.GetUserInfoRespond, int) {
	// redis
	zaplog.Info(uuid)
	rspString, err := myredis.GetKeyNilIsErr("user_info_" + uuid)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zaplog.Info(err.Error())
			var user model.UserInfo
			if res := mysql.GormDB.Where("uuid = ?", uuid).Find(&user); res.Error != nil {
				zaplog.Error(res.Error.Error())
				return constants.SYSTEM_ERROR, nil, -1
			}
			rsp := respond.GetUserInfoRespond{
				Uuid:      user.Uuid,
				Telephone: user.Telephone,
				Nickname:  user.Nickname,
				Avatar:    user.Avatar,
				Birthday:  user.Birthday,
				Email:     user.Email,
				Gender:    user.Gender,
				Signature: user.Signature,
				CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
				IsAdmin:   user.IsAdmin,
				Status:    user.Status,
			}
			return "获取用户信息成功", &rsp, 0
		} else {
			zaplog.Error(err.Error())
			return constants.SYSTEM_ERROR, nil, -1
		}
	}
	var rsp respond.GetUserInfoRespond
	if err := json.Unmarshal([]byte(rspString), &rsp); err != nil {
		zaplog.Error(err.Error())
	}
	return "获取用户信息成功", &rsp, 0
}

// SetAdmin 设置管理员
func (u *userInfoService) SetAdmin(uuidList []string, isAdmin int8) (string, int) {
	var users []model.UserInfo
	if res := mysql.GormDB.Where("uuid in (?)", uuidList).Find(&users); res.Error != nil {
		zaplog.Error(res.Error.Error())
		return constants.SYSTEM_ERROR, -1
	}
	for _, user := range users {
		user.IsAdmin = isAdmin
		if res := mysql.GormDB.Save(&user); res.Error != nil {
			zaplog.Error(res.Error.Error())
			return constants.SYSTEM_ERROR, -1
		}
	}
	return "设置管理员成功", 0
}
