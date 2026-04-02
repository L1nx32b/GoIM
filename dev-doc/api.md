userinfo
端点	方法	描述
/register	POST	用户注册
/login	POST	使用凭据进行标准登录
/user/smsLogin	POST	短信验证码登录
/user/updateUserInfo	POST	更新用户个人资料信息
/user/getUserInfo	POST	获取特定用户信息
/user/getUserInfoList	POST	获取分页用户列表（管理员）
/user/ableUsers	POST	启用已禁用的用户（管理员）
/user/disableUsers	POST	禁用用户（管理员）
/user/deleteUsers	POST	永久删除用户（管理员）
/user/setAdmin	POST	分配管理员权限（管理员）
/user/sendSmsCode	POST	发送短信验证码
/user/wsLogout	POST	WebSocket 登出

groupinfo
端点	方法	描述
/group/createGroup	POST	创建新群聊
/group/loadMyGroup	POST	获取当前用户创建的群组
/group/checkGroupAddMode	POST	检查加入群组的要求
/group/enterGroupDirectly	POST	无需审批直接加入公开群组
/group/leaveGroup	POST	退出群组
/group/dismissGroup	POST	解散拥有的群组
/group/getGroupInfo	POST	获取群组详情
/group/getGroupInfoList	POST	获取所有群组（管理员）
/group/deleteGroups	POST	删除群组（管理员）
/group/setGroupsStatus	POST	启用/禁用群组（管理员）
/group/updateGroupInfo	POST	更新群组详情
/group/getGroupMemberList	POST	获取群组成员列表
/group/removeGroupMembers	POST	从群组中移除成员

session
端点	方法	描述
/session/openSession	POST	初始化新的对话会话
/session/getUserSessionList	POST	获取用户的对话会话
/session/getGroupSessionList	POST	获取用户的群组会话
/session/deleteSession	POST	删除对话会话
/session/checkOpenSessionAllowed	POST	验证创建会话的权限

usercontact
端点	方法	描述
/contact/getUserList	POST	获取用户的联系人列表
/contact/loadMyJoinedGroup	POST	获取用户加入的群组
/contact/getContactInfo	POST	获取特定联系人详情
/contact/deleteContact	POST	从列表中删除联系人
/contact/applyContact	POST	发送好友请求
/contact/getNewContactList	POST	获取待处理的好友请求
/contact/passContactApply	POST	接受好友请求
/contact/refuseContactApply	POST	拒绝好友请求
/contact/blackContact	POST	拉黑联系人
/contact/cancelBlackContact	POST	解除拉黑联系人
/contact/getAddGroupList	POST	获取待处理的群组请求
/contact/blackApply	POST	拒绝群组加入请求

message
端点	方法	描述
/message/getMessageList	POST	获取私聊消息历史
/message/getGroupMessageList	POST	获取群聊消息历史
/message/uploadAvatar	POST	上传用户头像图片
/message/uploadFile	POST	上传文件附件

chatroom
端点	方法	描述
/chatroom/getCurContactListInChatRoom	POST	获取活动聊天室中的参与者

websocket
端点	方法	描述
/wss	GET	WebSocket 连接升级
/user/wsLogout	POST	终止 WebSocket 会话
