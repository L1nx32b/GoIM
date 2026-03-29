package api

import (
	"GoChatServer/internal/dto/request"
	"GoChatServer/internal/services/gorm"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/zaplog"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserList 获取联系人列表
func GetUserList(c *gin.Context) {
	var myUserListReq request.OwnlistRequest
	if err := c.BindJSON(&myUserListReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
	}
	message, userList, ret := gorm.UserContactService.GetUserList(myUserListReq.OwnerId)
	JsonBack(c, message, ret, userList)
}

// LoadMyjoinedGroup 获取我加入的群聊
func LoadMyjoinedGroup(c *gin.Context) {
	var loadMyJoinedGroupReq request.OwnlistRequest
	if err := c.BindJSON(&loadMyJoinedGroupReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
	}

	zaplog.Info(loadMyJoinedGroupReq.OwnerId)
	message, groupList, ret := gorm.UserContactService.LoadMyJoinedGroup(loadMyJoinedGroupReq.OwnerId)
	JsonBack(c, message, ret, groupList)
}

// GetContactInfo 获取联系人信息
func GetContactInfo(c *gin.Context) {
	var getContactInfoReq request.GetContactInfoRequest
	if err := c.BindJSON(&getContactInfoReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	log.Println(getContactInfoReq)
	message, contactInfo, ret := gorm.UserContactService.GetContactInfo(getContactInfoReq.ContactId)
	JsonBack(c, message, ret, contactInfo)
}

// DeleteContact 删除联系人
func DeleteContact(c *gin.Context) {
	var deleteContactReq request.DeleteContactRequest
	if err := c.BindJSON(&deleteContactReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.DeleteContact(deleteContactReq.OwnerId, deleteContactReq.ContactId)
	JsonBack(c, message, ret, nil)
}

// ApplyContact 申请添加联系人
func ApplyContact(c *gin.Context) {
	var applyContactReq request.ApplyContactRequest
	fmt.Println("applying")
	if err := c.BindJSON(&applyContactReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.ApplyContact(applyContactReq)
	JsonBack(c, message, ret, nil)
}

// GetNewContactList 获取新的联系人列表
func GetNewContactList(c *gin.Context) {
	var req request.OwnlistRequest
	if err := c.BindJSON(&req); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, data, ret := gorm.UserContactService.GetNewContactList(req.OwnerId)
	JsonBack(c, message, ret, data)
}

// PassContactApply 通过联系人申请
func PassContactApply(c *gin.Context) {
	var passContactApplyReq request.PassContactApplyRequest
	if err := c.BindJSON(&passContactApplyReq); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	messsage, ret := gorm.UserContactService.PassContactApply(passContactApplyReq.OwnerId, passContactApplyReq.ContactId)
	JsonBack(c, messsage, ret, nil)
}

// RefuseContactApply 拒绝联系人申请
func RefuseContactApply(c *gin.Context) {
	var refuseContactApply request.PassContactApplyRequest
	if err := c.BindJSON(&refuseContactApply); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.RefuseContactApply(refuseContactApply.OwnerId, refuseContactApply.ContactId)
	JsonBack(c, message, ret, nil)
}

// BlackContact 拉黑联系人
func BlackContact(c *gin.Context) {
	var blackContactReq request.BlackContactRequest
	if err := c.BindJSON(&blackContactReq); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.BlackContact(blackContactReq.OwnerId, blackContactReq.ContactId)
	JsonBack(c, message, ret, nil)
}

// CancelBlackContact 解除拉黑联系人
func CancelBlackContact(c *gin.Context) {
	var req request.BlackContactRequest
	if err := c.BindJSON(&req); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.CancelBlackContact(req.OwnerId, req.ContactId)
	JsonBack(c, message, ret, nil)
}

// GetAddGroupList 获取新的群聊申请列表
func GetAddGroupList(c *gin.Context) {
	var req request.AddGroupListRequest
	if err := c.BindJSON(&req); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, data, ret := gorm.UserContactService.GetAddGroupList(req.GroupId)
	JsonBack(c, message, ret, data)
}

// BlackApply 拉黑申请
func BlackApply(c *gin.Context) {
	var req request.BlackApplyRequest
	if err := c.BindJSON(&req); err != nil {
		zaplog.Error(err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": constants.SYSTEM_ERROR,
		})
		return
	}
	message, ret := gorm.UserContactService.BlackApply(req.OwnerId, req.ContactId)
	JsonBack(c, message, ret, nil)
}
