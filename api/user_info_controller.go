package api

import (
	"GoChatServer/internal/dto/request"
	"fmt"
	"net/http"

	"GoChatServer/pkg/constants"
	zlog "GoChatServer/pkg/zaplog"

	"GoChatServer/internal/services/gorm"

	"github.com/gin-gonic/gin"
)

// Login 登录
func Login(c *gin.Context) {
	// 从 Gin 上下文 c 中读取 JSON 请求体，将其绑定到 request.LoginRequest 结构体
	var req request.LoginRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(
			http.StatusOK,
			gin.H{
				"code":    "200",
				"message": constants.SYSTEM_ERROR,
			})
		return
	}
	message, userInfo, ret := gorm.UserInfoService.Login(req)
	//
	JsonBack(c, message, ret, userInfo)
}

// SmsLogin 验证码登录
func SmsLogin(c *gin.Context) {
	var req request.SmsLoginRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, userInfo, ret := gorm.UserInfoService.SmsLogin(req)
	JsonBack(c, message, ret, userInfo)
}

// SendSmsCode 发送短信验证码
func SendSmsCode(c *gin.Context) {
	// 从 Gin 上下文 c 中读取 JSON 请求体，将其绑定到 request.SendSmsCodeRequest 结构体
	var req request.SendSmsCodeRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(
			http.StatusOK,
			gin.H{
				"code":    "500",
				"message": constants.SYSTEM_ERROR,
			})
		return
	}
	message, ret := gorm.UserInfoService.SendSmsCode(req.Telephone)
	// 调用自定义的 JsonBack 函数，将 message、ret 以及 nil（可能表示无额外数据）包装成 JSON 格式返回给客户端。
	JsonBack(c, message, ret, nil)
}

// register 注册
func Register(c *gin.Context) {
	var registerReq request.RegisterRequest
	if err := c.BindJSON(&registerReq); err != nil {
		zlog.Error(err.Error())
		c.JSON(
			http.StatusOK,
			gin.H{
				"code":    "200",
				"message": constants.SYSTEM_ERROR,
			})
		return
	}
	fmt.Println(registerReq)
	message, userInfo, ret := gorm.UserInfoService.Register(registerReq)
	JsonBack(c, message, ret, userInfo)
}

// SetAdmin 设置管理员
func SetAdmin(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.SetAdmin(req.UuidList, req.IsAdmin)
	JsonBack(c, message, ret, nil)
}

// DeleteUsers 删除用户
func DeleteUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.DeleteUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

// GetUserInfo 获取用户信息
func GetUserInfo(c *gin.Context) {
	var req request.GetUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, userInfo, ret := gorm.UserInfoService.GetUserInfo(req.Uuid)
	JsonBack(c, message, ret, userInfo)
}

// UpdateUserInfo 修改用户信息
func UpdateUserInfo(c *gin.Context) {
	var req request.UpdateUserInfoRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.UpdateUserInfo(req)
	JsonBack(c, message, ret, nil)
}

// GetUserInfoList 获取用户列表
func GetUserInfoList(c *gin.Context) {
	var req request.GetUserInfoListRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, userList, ret := gorm.UserInfoService.GetUserInfoList(req.OwnerId)
	JsonBack(c, message, ret, userList)
}

// AbleUsers 启用用户
func AbleUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.AbleUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}

// DisableUsers 禁用用户
func DisableUsers(c *gin.Context) {
	var req request.AbleUsersRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserInfoService.DisableUsers(req.UuidList)
	JsonBack(c, message, ret, nil)
}
