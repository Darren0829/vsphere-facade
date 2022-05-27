package test

import (
	"vsphere-facade/app/cache"
	"vsphere-facade/app/logging"
	"vsphere-facade/config"
	"vsphere-facade/helper"
	"vsphere-facade/vsphere"
	vCache "vsphere-facade/vsphere/cache"
)

var vc35 = vsphere.Auth{
	Address:  "https://192.168.25.35",
	Username: "administrator@vsphere.locall",
	Password: "Zhu@88jie",
}

var vc30 = vsphere.Auth{
	Address:  "https://192.168.25.30",
	Username: "administrator@vsphere.local",
	Password: "1qaz@WSX",
}

var vcys = vsphere.Auth{
	Address:  "https://hosting51.3322.org:31444",
	Username: "administrator@leaptocloud.com",
	Password: "1qaz@WSX",
}

var vc *vsphere.VCenter

func init() {
	config.Setup()
	logging.Setup()
	helper.Setup()
	cache.Setup()
	vCache.Setup()
	vc = vsphere.Get(vc30)
}
