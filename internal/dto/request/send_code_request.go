package request

// 发送电话短信验证请求体
type SendSmsCodeRequest struct {
	Telephone string `json:"telephone"`
}

type SendEmailCodeRequest struct {
	Email string `json:"email"`
}
