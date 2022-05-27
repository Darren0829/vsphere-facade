package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere-facade/api/e"
	"vsphere-facade/api/security"
	"vsphere-facade/vsphere"
	"vsphere-facade/vsphere/protocol"
)

type NetworkQuery struct {
	DatacenterID string   `json:"datacenterId"`
	IDs          []string `json:"ids"`
}

// QueryNetworks
// @Summary      网络查询
// @Description  网络查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids           query     []string  false  "网络ID"
// @Param        datacenterId  query     string    false  "数据中心ID"
// @Success      200           {object}  e.Response{data=[]protocol.NetworkInfo}
// @Failure      400           {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401           {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500           {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/networks [get]
func QueryNetworks(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := NetworkQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&query)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	q := protocol.NetworkQuery{
		DatacenterID: query.DatacenterID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	networks := vc.QueryNetworks(q)
	if networks == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, networks)
	}
}
