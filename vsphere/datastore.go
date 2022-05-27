package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/helper/datastore"
	"vsphere-facade/helper/disk"
	"vsphere-facade/vsphere/protocol"
)

func (vc *VCenter) QueryDatastores(q protocol.DatastoreQuery) []protocol.DatastoreInfo {
	datastores := vc.queryDatastoresFromCache(q)
	if datastores != nil {
		logging.L().Debug("本次查询使用了缓存")
		return datastores
	}

	if len(q.IDs) > 0 {
		return vc.getDatastoresByIDs(q.IDs)
	} else if q.DatacenterID != "" {
		return vc.getDatastoresByDatacenterID(q.DatacenterID)
	} else {
		return vc.getDatastores()
	}
}

func (vc *VCenter) getDatastoresByIDs(IDs []string) []protocol.DatastoreInfo {
	var datastoreInfos []protocol.DatastoreInfo
	for _, ID := range IDs {
		moDatastore := datastore.GetMObject(vc.Api, ID)
		if moDatastore == nil {
			continue
		}
		datastoreInfo := vc.buildDatastoreInfo(*moDatastore, "")
		datastoreInfos = append(datastoreInfos, datastoreInfo)
	}
	return datastoreInfos
}

func (vc *VCenter) getDatastoresByDatacenterID(datacenterID string) []protocol.DatastoreInfo {
	moDatastores := datastore.GetByDatacenterID(vc.Api, datacenterID)
	if moDatastores == nil {
		return nil
	}
	var datastoreInfos []protocol.DatastoreInfo
	for _, moDatastore := range *moDatastores {
		datastoreInfo := vc.buildDatastoreInfo(moDatastore, datacenterID)
		datastoreInfos = append(datastoreInfos, datastoreInfo)
	}
	return datastoreInfos
}

func (vc *VCenter) getDatastores() []protocol.DatastoreInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var datastoreInfos []protocol.DatastoreInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moDatastores := datastore.GetByDatacenterID(vc.Api, datacenterID)
		if moDatastores == nil {
			continue
		}
		for _, moDatastore := range *moDatastores {
			datastoreInfo := vc.buildDatastoreInfo(moDatastore, datacenterID)
			datastoreInfos = append(datastoreInfos, datastoreInfo)
		}
	}

	vc.Cache.CacheDatastores(datastoreInfos)
	return datastoreInfos
}

func (vc *VCenter) buildDatastoreInfo(moDatastore mo.Datastore, datacenterID string) protocol.DatastoreInfo {
	var datastoreInfo protocol.DatastoreInfo
	datastoreInfo.DatacenterID = datacenterID
	datastoreInfo.ID = moDatastore.Reference().Value
	datastoreInfo.Name = moDatastore.Name

	summary := moDatastore.Summary
	datastoreInfo.Type = summary.Type
	datastoreInfo.Accessible = summary.Accessible
	datastoreInfo.Capacity = summary.Capacity
	datastoreInfo.FreeSpace = summary.FreeSpace
	datastoreInfo.Uncommitted = summary.Uncommitted
	datastoreInfo.SupportDiskType = disk.GetFormats(summary.Type)
	return datastoreInfo
}

func (vc *VCenter) queryDatastoresFromCache(q protocol.DatastoreQuery) []protocol.DatastoreInfo {
	cache := vc.Cache.GetDatastores()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var datastores []protocol.DatastoreInfo
		for _, Datastore := range cache {
			if utils.SliceContain(q.IDs, Datastore.ID) {
				datastores = append(datastores, Datastore)
			}
		}
		return datastores
	} else if q.DatacenterID != "" {
		var datastores []protocol.DatastoreInfo
		for _, Datastore := range cache {
			if Datastore.DatacenterID == q.DatacenterID {
				datastores = append(datastores, Datastore)
			}
		}
		return datastores
	} else {
		return cache
	}
}
