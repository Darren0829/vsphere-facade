package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"vsphere_api/api/router"
	"vsphere_api/api/security"
	"vsphere_api/app/cache"
	"vsphere_api/app/logging"
	"vsphere_api/config"
	"vsphere_api/db"
	"vsphere_api/helper"
	"vsphere_api/startup"
	vCache "vsphere_api/vsphere/cache"
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
