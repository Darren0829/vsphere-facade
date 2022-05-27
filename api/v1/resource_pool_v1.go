package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	"vsphere_api/vsphere"
	"vsphere_api/vsphere/protocol"
)

type ResourcePoolQuery struct {
	DatacenterID string
	ClusterID    string
	HostID       string
	IDs          []string
}

// QueryResourcePools
// @Summary      资源池查询
// @Description  资源池查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids           query     []string  false  "资源池ID"
// @Param        datacenterId  query     string    false  "数据中心ID"
// @Param        clusterId     query     string    false  "集群ID"
// @Param        hostId        query     string    false  "主机ID"
// @Success      200           {object}  e.Response{data=[]protocol.ResourcePoolInfo}
// @Failure      400           {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401           {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500           {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/resource_pools [get]
func QueryResourcePools(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := ResourcePoolQuery{}
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

	q := protocol.ResourcePoolQuery{
		DatacenterID: query.DatacenterID,
		ClusterID:    query.ClusterID,
		HostID:       query.HostID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	resourcePools := vc.QueryResourcePools(q)
	if resourcePools == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, resourcePools)
	}
}
