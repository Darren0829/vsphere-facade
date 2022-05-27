package e

var Message = map[string]string{
	Success:       "成功",
	Accepted:      "请求已接收",
	SystemError:   "错误",
	Unauthorized:  "需要认证Token",
	ConnectFailed: "连接失败",
	NotEnabled:    "未开启配置",
	NotFound:      "请求地址不存在",
	VMNotFound:    "虚拟机不存在",
	TokenInvalid:  "token无效",
	TokenExpired:  "token已过期",
	"ServerFaultCode: Cannot complete login due to an incorrect user name or password.": "无法连接VC，账号或密码错误",
}

func GetMessage(code string) string {
	msg, ok := Message[code]
	if ok {
		return msg
	}
	return code
}
