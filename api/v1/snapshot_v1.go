package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere-facade/api/e"
	"vsphere-facade/api/security"
	"vsphere-facade/vsphere"
	"vsphere-facade/vsphere/workerpool"
	"vsphere-facade/vsphere/workerpool/taskreceiver"
)

// CreateSnapshot
// @Summary      创建快照
// @Description  创建快照
// @Tags         快照
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "创建参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/{id}/snapshots [post]
func CreateSnapshot(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeDeployment, p)
	go func() {
		defer taskreceiver.Done(res.RequestID)
		vsphere.Get(auth)
	}()
	r.ResponseOk(http.StatusAccepted, e.Accepted, res)
}

// DeleteSnapshot
// @Summary      删除快照
// @Description  删除快照
// @Tags         快照
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "创建参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/{id}/snapshots [delete]
func DeleteSnapshot(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeDeployment, p)
	go func() {
		defer taskreceiver.Done(res.RequestID)
		vsphere.Get(auth)
	}()
	r.ResponseOk(http.StatusAccepted, e.Accepted, res)
}
