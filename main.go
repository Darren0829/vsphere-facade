package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"vsphere-facade/api/router"
	"vsphere-facade/api/security"
	"vsphere-facade/app/cache"
	"vsphere-facade/app/logging"
	"vsphere-facade/config"
	"vsphere-facade/db"
	"vsphere-facade/helper"
	"vsphere-facade/startup"
	vCache "vsphere-facade/vsphere/cache"
)

func init() {
	config.Setup()
	logging.Setup()
	security.Setup()
	helper.Setup()
	cache.Setup()
	vCache.Setup()
	db.Setup()
}

// @title        QXP vSphere API
// @version      1.0
// @description  vmware vsphere api

// @host      localhost:8829
// @BasePath  /api

// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        token
func main() {
	defer logging.Sync()
	gin.SetMode(config.G.Server.Mode)
	r := router.InitRouter()
	initSwagger(r)
	go startup.Run()
	_ = r.Run(fmt.Sprintf(":%d", config.G.Server.Port))
}

var swagHandler gin.HandlerFunc

func initSwagger(r *gin.Engine) {
	if swagHandler != nil {
		r.GET("/swagger/*any", swagHandler)
	}
}
