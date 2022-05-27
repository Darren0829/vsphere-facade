package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	"vsphere_api/app/logging"
	"vsphere_api/vsphere"
	"vsphere_api/vsphere/protocol"
)

type DatastoreQuery struct {
	DatacenterID string   `form:"datacenterId"`
	IDs          []string `form:"ids"`
}

// QueryDatastores
// @Summary      存储查询
// @Description  存储查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids           query     []string  false  "存储ID"
// @Param        datacenterId  query     string    false  "数据中心ID"
// @Success      200           {object}  e.Response{data=[]protocol.DatastoreInfo}
// @Failure      400           {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401           {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500           {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/datastores [get]
func QueryDatastores(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := DatastoreQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logging.L().Error("解析请求参数时发生错误", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&query)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	q := protocol.DatastoreQuery{
		DatacenterID: query.DatacenterID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	datastores := vc.QueryDatastores(q)
	if datastores == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, datastores)
	}
}
