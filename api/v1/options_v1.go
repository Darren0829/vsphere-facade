package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere-facade/api/e"
	"vsphere-facade/api/security"
	"vsphere-facade/vsphere"
	"vsphere-facade/vsphere/protocol"
)

// QueryDatacenters
// @Summary      数据中心查询
// @Description  数据中心查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids  query     string  false  "数据中心ID"
// @Success      200  {object}  e.Response{data=[]protocol.DatacenterInfo}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/datacenters [get]
func QueryDatacenters44(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := DatacenterQuery{}
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
	q := protocol.DatacenterQuery{
		IDs: query.IDs,
	}
	datacenters := vc.QueryDatacenters(q)
	if datacenters == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, datacenters)
	}
}

func GetDiskModes(c *gin.Context) {
}

func GetDiskFormatOptions(c *gin.Context) {

}
