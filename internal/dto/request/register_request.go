package request

// 手机注册请求体
type RegisterRequest struct {
	Telephone string `json:"telephone"`
	Password  string `json:"password"`
	Nickname  string `json:"nickname"`
	SmsCode   string `json:"sms_code"`
}

// 邮箱注册请求体
type EmailRegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Nickname  string `json:"nickname"`
	EmailCode string `json:"emailcode"`
}
