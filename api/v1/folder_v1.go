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

type FolderQuery struct {
	DatacenterID string   `form:"datacenterId"`
	FolderID     string   `form:"folderId"`
	IDs          []string `form:"ids"`
}

// QueryFolders
// @Summary      文件夹查询
// @Description  文件夹查询
// @Tags         基础设施
// @Accept       json
// @Produce      json
// @Param        ids           query     []string  false  "文件夹ID"
// @Param        datacenterId  query     string    false  "数据中心ID"
// @Param        folderID      query     string    false  "文件夹ID"
// @Success      200           {object}  e.Response{data=[]protocol.FolderInfo}
// @Failure      400           {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401           {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500           {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/folders [get]
func QueryFolders(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := FolderQuery{}
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
	q := protocol.FolderQuery{
		DatacenterID: query.DatacenterID,
		FolderID:     query.FolderID,
		IDs:          query.IDs,
	}
	folders := vc.QueryFolders(q)
	if folders == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, folders)
	}
}
