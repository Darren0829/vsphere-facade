package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	"vsphere_api/vsphere"
	"vsphere_api/vsphere/protocol"
)

type HostQuery struct {
	DatacenterID string   `form:"datacenterId"`
	ClusterID    string   `form:"clusterId"`
	IDs          []string `form:"ids"`
}

// QueryHosts
// @Summary      主机查询
// @Description  主机查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Success      200  {object}  e.Response{data=[]protocol.OSFamilyInfo}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/hosts [get]
func QueryHosts(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := HostQuery{}
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

	q := protocol.HostQuery{
		DatacenterID: query.DatacenterID,
		ClusterID:    query.ClusterID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	hosts := vc.QueryHosts(q)
	if hosts == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, hosts)
	}
}

// GetHostOSFamilies
// @Summary      主机支持的操作系统
// @Description  主机支持的操作系统
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        hostID  path      string  true  "主机ID"
// @Success      200     {object}  e.Response{data=[]protocol.OSFamilyInfo}
// @Failure      400     {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401     {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500     {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/hosts/{hostId}/os_families [get]
func GetHostOSFamilies(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)
	hostID := c.Param("hostID")

	var vc = vsphere.Get(auth)
	OSFamilyInfos := vc.GetHostOSFamilies(hostID)
	if OSFamilyInfos == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, OSFamilyInfos)
	}
}
