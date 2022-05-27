package vsphere

import (
	"fmt"
	"time"
	"vsphere_api/app/logging"
	"vsphere_api/config"
	"vsphere_api/helper"
	"vsphere_api/vsphere/cache"
	"vsphere_api/vsphere/protocol"
	"vsphere_api/vsphere/workerpool"
)

type Auth struct {
	Address  string `json:"address" valid:"Required"`
	Username string `json:"username" valid:"Required"`
	Password string `json:"password" valid:"Required"`
}

type VCenter struct {
	Api   *helper.API
	Cache *cache.VCCache
}

func Get(a Auth) *VCenter {
	vc := VCenter{}
	api := helper.Init(a.Address, a.Username, a.Password)
	vc.Api = api
	vc.Cache = &cache.VCCache{
		VCID: api.ID,
	}
	return &vc
}

func (vc *VCenter) CreateCache() {
	if config.G.Vsphere.Cache.Enable {
		logging.L().Debug(fmt.Sprintf("为VCenter[%s]创建缓存数据开始", vc.Cache.VCID))
		vc.createCache()
		vc.CreateCacheTicker()
		logging.L().Debug(fmt.Sprintf("为VCenter[%s]创建缓存数据完成", vc.Cache.VCID))
	} else {
		logging.L().Debug("未开启缓存配置")
	}
}

func (vc *VCenter) createCache() {
	vc.Cache.CleanAll()
	vc.getDatacenters()
	vc.getDatastores()
	vc.getResourcePools()
	vc.getFolders()
	vc.getClusters()
	vc.getHosts()
	vc.getNetworks()
	vc.QueryStoragePolicies(protocol.StoragePolicyQuery{})
	vc.getOSFamilies()
}

func (vc *VCenter) AddTask(t workerpool.WorkerType, task func()) error {
	return workerpool.AddTask(vc.Api.ID, t, task)
}

func (vc *VCenter) CreateCacheTicker() {
	refreshDuration := config.G.Vsphere.Cache.RefreshDuration
	if refreshDuration > 0 {
		logging.L().Debugf("为VC[%s]创建缓存刷新定时器，刷新间隔设置为: %dm", vc.Api.ID, refreshDuration)
		_, exists := vc.Cache.Get(cache.RefreshTicker)
		if exists {
			logging.L().Debug("定时器已经存在，跳过")
			return
		}
		ticker := time.NewTicker(time.Minute * time.Duration(refreshDuration))
		vc.Cache.Set(cache.RefreshTicker, ticker)
	} else {
		logging.L().Debugf("[refreshDuration: %d]刷新间隔设置小于0，跳过缓存刷新定时器创建", refreshDuration)
	}
}

func (vc *VCenter) StopTicker() {
	logging.L().Debugf("停止VC[%s]缓存刷新定时器", vc.Api.ID)
	ticker, exists := vc.Cache.Get(cache.RefreshTicker)
	if exists {
		ticker.(*time.Ticker).Stop()
		vc.Cache.Clean(cache.RefreshTicker)
		logging.L().Debugf("VC[%s]缓存刷新定时器已停止", vc.Api.ID)
	} else {
		logging.L().Debug("定时器不存在，跳过")
	}
}
