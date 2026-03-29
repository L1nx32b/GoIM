package request

// 手机验证码登录
type SmsLoginRequest struct {
	Telephone string `json:"telephone"`
	SmsCode   string `json:"smscode"`
}

// 邮箱验证码登录
type EmailCodeLoginRequest struct {
	Email     string `json:"email"`
	EmailCode string `json:"emailcode"`
}
