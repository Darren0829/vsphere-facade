package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere-facade/api/e"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/vsphere/protocol"
)

// ReceiveCallBackData
// @Summary      测试http回调
// @Description  测试http回调
// @Tags         测试
// @Accept       json
// @Produce      json
// @Param        c  body  protocol.CallbackRes  true  "回调结果"
// @Success      200
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/test_call_back [post]
func ReceiveCallBackData(c *gin.Context) {
	r := e.Gin{C: c}
	body := protocol.CallbackRes{}
	err := c.ShouldBind(&body)
	if err != nil {
		logging.L().Error("解析请求参数时发生错误", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}
	logging.L().Debug("回调数据: ", utils.ToJson(body))
}
