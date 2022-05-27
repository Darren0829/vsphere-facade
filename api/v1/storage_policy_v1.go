package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	"vsphere_api/vsphere"
	"vsphere_api/vsphere/protocol"
)

type StoragePolicyQuery struct {
}

// QueryStoragePolies
// @Summary      存储策略查询
// @Description  存储策略查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Success      200  {object}  e.Response{data=[]protocol.StoragePolicyInfo}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/storage_policies [get]
func QueryStoragePolies(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := StoragePolicyQuery{}
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

	var vc = vsphere.Get(auth)
	policies := vc.QueryStoragePolicies(protocol.StoragePolicyQuery{})
	if policies == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, policies)
	}
}
