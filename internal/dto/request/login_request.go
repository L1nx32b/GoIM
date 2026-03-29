package request

// 前端手机登录请求体
type LoginRequest struct {
	Telephone string `json:"telephone"`
	Password  string `json:"password"`
}

// 前端邮箱登录请求体
type EmailLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
