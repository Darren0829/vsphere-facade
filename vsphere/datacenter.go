package vsphere

import (
	"github.com/vmware/govmomi/object"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/helper/datacenter"
	"vsphere-facade/vsphere/protocol"
)

func (vc *VCenter) QueryDatacenters(q protocol.DatacenterQuery) []protocol.DatacenterInfo {
	dcs := vc.queryDatacentersFromCache(q)
	if dcs != nil {
		logging.L().Debug("本次查询使用了缓存")
		return dcs
	}

	if len(q.IDs) > 0 {
		return vc.getDatacenterByIDs(q.IDs)
	} else {
		return vc.getDatacenters()
	}
}

func (vc *VCenter) getDatacenterByIDs(IDs []string) []protocol.DatacenterInfo {
	oDatacenters := datacenter.GetAll(vc.Api)
	if oDatacenters == nil {
		return nil
	}

	var datacenterInfos []protocol.DatacenterInfo
	for _, oDatacenter := range oDatacenters {
		if utils.SliceContain(IDs, oDatacenter.Reference().Value) {
			datacenterInfo := vc.buildDatacenterInfo(oDatacenter)
			datacenterInfos = append(datacenterInfos, *datacenterInfo)
		}
	}
	return datacenterInfos
}

func (vc *VCenter) getDatacenters() []protocol.DatacenterInfo {
	datacenterInfos := vc.Cache.GetDatacenters()
	if datacenterInfos != nil {
		return datacenterInfos
	}
	oDatacenters := datacenter.GetAll(vc.Api)
	if oDatacenters == nil {
		return nil
	}

	for _, oDatacenter := range oDatacenters {
		datacenterInfo := vc.buildDatacenterInfo(oDatacenter)
		datacenterInfos = append(datacenterInfos, *datacenterInfo)
	}

	vc.Cache.CacheDatacenters(datacenterInfos)
	return datacenterInfos
}

func (vc *VCenter) buildDatacenterInfo(oDatacenter *object.Datacenter) *protocol.DatacenterInfo {
	var datacenterInfo protocol.DatacenterInfo
	datacenterInfo.ID = oDatacenter.Reference().Value
	datacenterInfo.Name = oDatacenter.Name()
	return &datacenterInfo
}

func (vc *VCenter) queryDatacentersFromCache(q protocol.DatacenterQuery) []protocol.DatacenterInfo {
	cache := vc.Cache.GetDatacenters()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var Datacenters []protocol.DatacenterInfo
		for _, Datacenter := range cache {
			if utils.SliceContain(q.IDs, Datacenter.ID) {
				Datacenters = append(Datacenters, Datacenter)
			}
		}
		return Datacenters
	} else {
		return cache
	}
}
