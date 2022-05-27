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

type ClusterQuery struct {
	DatacenterID string   `form:"datacenterId"`
	IDs          []string `form:"ids"`
}

// QueryClusters
// @Summary      集群查询
// @Description  集群查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids           query     []string  false  "集群ID"
// @Param        datacenterId  query     string    false  "数据中心ID"
// @Success      200           {object}  e.Response{data=[]protocol.ClusterInfo}
// @Failure      400           {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401           {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500           {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/clusters [get]
func QueryClusters(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := ClusterQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		logging.L().Error("请求参数解析时发生错误", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&query)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	var vc = vsphere.Get(auth)
	q := protocol.ClusterQuery{
		DatacenterID: query.DatacenterID,
		IDs:          query.IDs,
	}
	clusters := vc.QueryClusters(q)
	if clusters == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, clusters)
	}
}

// GetClusterOSFamilies
// @Summary      集群支持的操作系统
// @Description  集群支持的操作系统
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        clusterID  path      string  true  "集群ID"
// @Success      200        {object}  e.Response{data=[]protocol.OSFamilyInfo}
// @Failure      400        {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401        {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500        {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/clusters/{clusterId}/os_families [get]
func GetClusterOSFamilies(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)
	clusterID := c.Param("clusterID")

	var vc = vsphere.Get(auth)
	OSFamilyInfos := vc.GetComputerResourceOSFamilies(clusterID)
	if OSFamilyInfos == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, OSFamilyInfos)
	}
}
