package sms

import (
	"GoChatServer/internal/config"
	myredis "GoChatServer/internal/services/redis"
	"GoChatServer/pkg/constants"
	"GoChatServer/pkg/util/random"
	zlog "GoChatServer/pkg/zaplog"
	"fmt"
	"strconv"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

var smsClient *dysmsapi20170525.Client

// 使用AK&SK初始化账号Client
func createClient() (result *dysmsapi20170525.Client, err error) {
	accessKeyID := config.GetConfig().AccessKeyID
	accessKeySecret := config.GetConfig().AccessKeySecret

	if smsClient == nil {
		config := &openapi.Config{
			AccessKeyId:     tea.String(accessKeyID),
			AccessKeySecret: tea.String(accessKeySecret),
		}
		config.Endpoint = tea.String("dysmsapi.aliyuncs.com")
		smsClient, err = dysmsapi20170525.NewClient(config)
	}
	return smsClient, err
}

// 短信验证服务
func VerificationCode(telephone string) (string, int) {
	client, err := createClient()
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	key := "auth_code_" + telephone
	code, err := myredis.GetKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}

	if code != "" {
		// 直接返回, 验证码还没有过期, 用户应该输入验证码
		message := "目前还不能发送验证码, 请输入已发送验证码"
		zlog.Info(message)
		return message, -2
	}

	// 验证码过期, 重新生成
	code = strconv.Itoa(random.GetRandomInt(6))
	fmt.Println(code)

	err = myredis.SetKeyEx(key, code, time.Minute) // 一分钟有效
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	sendSmsRequest := &dysmsapi20170525.SendSmsRequest{
		SignName:      tea.String("阿里云短信测试"),
		TemplateCode:  tea.String("SMS_154950909"), // 短信模板
		PhoneNumbers:  tea.String(telephone),
		TemplateParam: tea.String("{\"code\":\"" + code + "\"}"),
	}

	runtime := &util.RuntimeOptions{}
	// 目前使用的是测试专用签名，签名必须是“阿里云短信测试”，模板code为“SMS_154950909”
	rsp, err := client.SendSmsWithOptions(sendSmsRequest, runtime)
	if err != nil {
		zlog.Error(err.Error())
		return constants.SYSTEM_ERROR, -1
	}
	zlog.Info(*util.ToJSONString(rsp))
	return "验证码发送成功，请及时在对应电话查收短信", 0
}
