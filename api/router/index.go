package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/vsphere/workerpool/taskreceiver"
)

func Index(c *gin.Context) {
	r := e.Gin{C: c}
	req := requests()
	r.C.String(http.StatusOK, "This is QXP vSphere API \n"+req)
}

func requests() string {
	all := taskreceiver.GetReceivedReq()
	s := " 请求 ｜ 参数"
	for id, req := range all {
		s += fmt.Sprintf("\n %s | %s", id, req)
	}
	return s
}
