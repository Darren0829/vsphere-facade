package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	"vsphere_api/app/logging"
	"vsphere_api/config"
	"vsphere_api/vsphere"
)

type CleanCacheKey struct {
	Keys *[]string
}

// CleanCache
// @Summary      清除缓存
// @Description  清除缓存
// @Tags         缓存
// @Accept       json
// @Produce      json
// @Param        c    body      CleanCacheKey  false  "需要清除的缓存Key，空为全部"
// @Success      200  {object}  e.Response
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/caches [delete]
func CleanCache(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := CleanCacheKey{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数时发生错误", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	var vc = vsphere.Get(auth)
	if p.Keys != nil {
		vc.Cache.Clean(*p.Keys...)
	} else {
		vc.Cache.CleanAll()
	}
	r.ResponseOk(http.StatusOK, e.Success, nil)
}

// CreateCache
// @Summary      创建缓存
// @Description  创建缓存
// @Tags         缓存
// @Accept       json
// @Produce      json
// @Success      200  {object}  e.Response
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/caches [post]
func CreateCache(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	if config.G.Vsphere.Cache.Enable {
		var vc = vsphere.Get(auth)
		go vc.CreateCache()
		r.ResponseOk(http.StatusAccepted, e.Accepted, nil)
	} else {
		r.ResponseOk(http.StatusBadRequest, e.NotEnabled, nil)
	}
}
