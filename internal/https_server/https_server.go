package https_server

import (
	api "GoChatServer/api"
	"GoChatServer/internal/config"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
)

var GE *gin.Engine

func init() {
	// cors跨域处理
	GE = gin.Default()
	corsConfig := cors.DefaultConfig()                                                              // 获取CORS中间件的默认配置
	corsConfig.AllowOrigins = []string{"*"}                                                         // 设置允许的源(Origin)列表, "*"接受任何源的请求
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}                   // 设置允许的HTTP请求
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"} // 设置允许在跨域请求中携带的HTTP头信息
	GE.Use(cors.New(corsConfig))

	// GE.Use(ssl.TlsHandler(config.GetConfig().MainConfig.Host, config.GetConfig().MainConfig.Port))
	GE.Static("/static/avatars", config.GetConfig().StaticAvatarPath)
	GE.Static("/static/files", config.GetConfig().StaticFilePath)
	GE.POST("/login", api.Login)
	GE.POST("/register", api.Register)
	GE.POST("/user/updateUserInfo", api.UpdateUserInfo)
	GE.POST("/user/getUserInfoList", api.GetUserInfoList)
	GE.POST("/user/ableUsers", api.AbleUsers)
	GE.POST("/user/getUserInfo", api.GetUserInfo)
	GE.POST("/user/disableUsers", api.DisableUsers)
	GE.POST("/user/deleteUsers", api.DeleteUsers)
	GE.POST("/user/setAdmin", api.SetAdmin)
	GE.POST("/user/sendSmsCode", api.SendSmsCode)
	GE.POST("/user/smsLogin", api.SmsLogin)
	GE.POST("/group/createGroup", api.CreateGroup)
	GE.POST("/group/loadMyGroup", api.LoadMyGroup)
	GE.POST("/group/checkGroupAddMode", api.CheckGroupAddMode)
	GE.POST("/group/enterGroupDirectly", api.EnterGroupDirectly)
	GE.POST("/group/leaveGroup", api.LeaveGroup)
	GE.POST("/group/dismissGroup", api.DismissGroup)
	GE.POST("/group/getGroupInfo", api.GetGroupInfo)
	GE.POST("/group/getGroupInfoList", api.GetGroupInfoList)
	GE.POST("/group/deleteGroups", api.DeleteGroups)
	GE.POST("/group/setGroupsStatus", api.SetGroupsStatus)
	GE.POST("/group/updateGroupInfo", api.UpdateGroupInfo)
	GE.POST("/group/getGroupMemberList", api.GetGroupMemberList)
	GE.POST("/group/removeGroupMembers", api.RemoveGroupMembers)
	GE.POST("/session/openSession", api.OpenSession)
	GE.POST("/session/getUserSessionList", api.GetUserSessionList)
	GE.POST("/session/getGroupSessionList", api.GetGroupSessionList)
	GE.POST("/session/deleteSession", api.DeleteSession)
	GE.POST("/session/checkOpenSessionAllowed", api.CheckOpenSessionAllowed)
	GE.POST("/contact/getUserList", api.GetUserList)
	GE.POST("/contact/loadMyJoinedGroup", api.LoadMyjoinedGroup)
	GE.POST("/contact/getContactInfo", api.GetContactInfo)
	GE.POST("/contact/deleteContact", api.DeleteContact)
	GE.POST("/contact/applyContact", api.ApplyContact)
	GE.POST("/contact/getNewContactList", api.GetNewContactList)
	GE.POST("/contact/passContactApply", api.PassContactApply)
	GE.POST("/contact/blackContact", api.BlackContact)
	GE.POST("/contact/cancelBlackContact", api.CancelBlackContact)
	GE.POST("/contact/getAddGroupList", api.GetAddGroupList)
	GE.POST("/contact/refuseContactApply", api.RefuseContactApply)
	GE.POST("/contact/blackApply", api.BlackApply)
	GE.POST("/message/getMessageList", api.GetMessageList)
	GE.POST("/message/getGroupMessageList", api.GetGroupMessageList)
	GE.POST("/message/uploadAvatar", api.UploadAvatar)
	GE.POST("/message/uploadFile", api.UploadFile)
	GE.POST("/chatroom/getCurContactListInChatRoom", api.GetCurContactListInChatRoom)
	GE.GET("/wss", api.WsLogin)
	GE.POST("/user/wsLogout", api.WsLogout)
}
