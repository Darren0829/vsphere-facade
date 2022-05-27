package router

import (
	"embed"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security"
	v1 "vsphere_api/api/v1"
	_ "vsphere_api/docs"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	r.NoRoute(e.HandlerNotFound)
	r.NoMethod(e.HandlerNotFound)
	r.Use(e.ErrHandler)
	favicon(r)

	r.GET("/", Index)
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{"status": "on"})

	})
	r.POST("/api/token", security.GetToken)

	apiV1 := r.Group("/api/v1")
	apiV1.Use(security.Verify())
	{
		apiV1.GET("/datacenters", v1.QueryDatacenters)
		apiV1.GET("/clusters", v1.QueryClusters)
		apiV1.GET("/clusters/:clusterID/os_families", v1.GetClusterOSFamilies)
		apiV1.GET("/hosts", v1.QueryHosts)
		apiV1.GET("/hosts/:hostID/os_families", v1.GetHostOSFamilies)
		apiV1.GET("/networks", v1.QueryNetworks)
		apiV1.GET("/datastores", v1.QueryDatastores)
		apiV1.GET("/resource_pools", v1.QueryResourcePools)
		apiV1.GET("/storage_policies", v1.QueryStoragePolies)
		apiV1.GET("/folders", v1.QueryFolders)
		apiV1.GET("/templates", v1.QueryTemplates)

		// 虚拟机
		apiV1.GET("/virtual_machines", v1.QueryVirtualMachines)
		apiV1.POST("/virtual_machines", v1.CreateVirtualMachine)
		apiV1.DELETE("/virtual_machines", v1.DeleteVirtualMachine)
		apiV1.POST("/virtual_machines/power_on", v1.VirtualMachinePowerOn)
		apiV1.POST("/virtual_machines/power_off", v1.VirtualMachinePowerOff)
		apiV1.POST("/virtual_machines/shutdown", v1.VirtualMachineShutdown)
		apiV1.POST("/virtual_machines/rename", v1.VirtualMachineRename)
		apiV1.POST("/virtual_machines/reconfigure", v1.ModifyVirtualMachineConfigure)
		apiV1.POST("/virtual_machines/reconfigure_disk", v1.ReconfigureVirtualMachineDisk)
		apiV1.POST("/virtual_machines/reconfigure_nic", v1.ReconfigureVirtualMachineNic)
		apiV1.POST("/virtual_machines/description", v1.VirtualMachineDescript)

		// 缓存
		apiV1.DELETE("caches", v1.CleanCache)
		apiV1.POST("caches", v1.CreateCache)

		// 测试
		apiV1.POST("/test_call_back", v1.ReceiveCallBackData)
	}
	return r
}

//go:embed static
var efs embed.FS

func favicon(e *gin.Engine) {
	f, err := fs.Sub(efs, "static")
	if err != nil {
		panic(err)
	}
	handler := func(c *gin.Context) {
		c.FileFromFS("favicon.ico", http.FS(f))
	}
	e.GET("/favicon.ico", handler)
	e.HEAD("/favicon.ico", handler)
}
