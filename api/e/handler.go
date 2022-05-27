package e

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime/debug"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils/stringutils"
)

func HandlerNotFound(c *gin.Context) {
	message := fmt.Sprintf("请求地址[%s %s]不存在", c.Request.Method, c.Request.URL.String())
	c.JSON(http.StatusNotFound, Response{
		Code:    NotFound,
		Message: message,
		Data:    nil,
	})
	return
}

func ErrHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			var code string
			var message string
			switch e := r.(type) {
			case error:
				code = SystemError
				message = GetMessage(e.Error())
			case string:
				code = SystemError
				message = stringutils.EPTThen(GetMessage(e), GetMessage(code))
			default:
				code = SystemError
				message = GetMessage(code)
			}
			logging.L().Error(string(debug.Stack()))
			c.JSON(http.StatusInternalServerError, Response{
				Code:    code,
				Message: message,
				Data:    nil,
			})
			c.Abort()
		}
	}()
	c.Next()
}
