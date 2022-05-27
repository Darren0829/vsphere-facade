package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/helper/resourcepool"
	"vsphere-facade/vsphere/protocol"
)

const (
	ResourcePoolParentTypeCluster      = "CLUSTER"
	ResourcePoolParentTypeHost         = "HOST"
	ResourcePoolParentTypeResourcePool = "RESOURCE_POOL"
)

// QueryResourcePools 查询集群和主机下的资源池，virtual app的资源池在virtual app中
func (vc *VCenter) QueryResourcePools(q protocol.ResourcePoolQuery) []protocol.ResourcePoolInfo {
	resourcePools := vc.queryResourcePoolsFromCache(q)
	if resourcePools != nil {
		logging.L().Debug("本次查询使用了缓存")
		return resourcePools
	}

	if len(q.IDs) > 0 {
		return vc.getResourcePoolsByIDs(q.IDs)
	} else if q.ClusterID != "" || q.HostID != "" {
		return vc.getResourcePoolsByClusterIDOrHostID(q.ClusterID, q.HostID)
	} else if q.DatacenterID != "" {
		return vc.getResourcePoolsByDatacenterID(q.DatacenterID)
	} else {
		return vc.getResourcePools()
	}
}

func (vc *VCenter) getResourcePoolsByIDs(IDs []string) []protocol.ResourcePoolInfo {
	var resourcePoolInfos []protocol.ResourcePoolInfo
	for _, ID := range IDs {
		moResourcePool := resourcepool.GetMObject(vc.Api, ID)
		if moResourcePool == nil {
			return nil
		}
		resourcePoolInfo := vc.buildResourcePoolInfo(*moResourcePool, "")
		resourcePoolInfos = append(resourcePoolInfos, resourcePoolInfo)
	}
	return resourcePoolInfos
}

func (vc *VCenter) getResourcePoolsByDatacenterID(datacenterID string) []protocol.ResourcePoolInfo {
	moPools := resourcepool.GetByDatacenterID(vc.Api, datacenterID)
	if moPools == nil {
		return nil
	}
	var resourcePoolInfos []protocol.ResourcePoolInfo
	for _, moPool := range *moPools {
		resourcePoolInfo := vc.buildResourcePoolInfo(moPool, datacenterID)
		resourcePoolInfos = append(resourcePoolInfos, resourcePoolInfo)
	}
	return resourcePoolInfos
}

func (vc *VCenter) getResourcePoolsByClusterIDOrHostID(clusterID string, hostID string) []protocol.ResourcePoolInfo {
	var resourcePoolInfos []protocol.ResourcePoolInfo
	if clusterID != "" {
		moPools := resourcepool.GetByClusterID(vc.Api, clusterID)
		if moPools != nil {
			for _, moPool := range *moPools {
				resourcePoolInfo := vc.buildResourcePoolInfo(moPool, "")
				resourcePoolInfos = append(resourcePoolInfos, resourcePoolInfo)
			}
		}
	}
	if hostID != "" {
		moPools := resourcepool.GetByHostID(vc.Api, hostID)
		if moPools != nil {
			for _, moPool := range *moPools {
				resourcePoolInfo := vc.buildResourcePoolInfo(moPool, "")
				resourcePoolInfos = append(resourcePoolInfos, resourcePoolInfo)
			}
		}
	}
	return resourcePoolInfos
}

func (vc *VCenter) getResourcePools() []protocol.ResourcePoolInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var resourcePoolInfos []protocol.ResourcePoolInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moPools := resourcepool.GetByDatacenterID(vc.Api, datacenterID)
		if moPools == nil {
			continue
		}
		for _, moPool := range *moPools {
			resourcePoolInfo := vc.buildResourcePoolInfo(moPool, datacenterID)
			resourcePoolInfos = append(resourcePoolInfos, resourcePoolInfo)
		}
	}

	vc.Cache.CacheResourcePools(resourcePoolInfos)
	return resourcePoolInfos
}

func (vc *VCenter) buildResourcePoolInfo(moPool mo.ResourcePool, datacenterID string) protocol.ResourcePoolInfo {
	var resourcePoolInfo protocol.ResourcePoolInfo
	resourcePoolInfo.DatacenterID = datacenterID
	resourcePoolInfo.ID = moPool.Reference().Value
	resourcePoolInfo.Name = moPool.Name
	resourcePoolInfo.AvailableCpu = moPool.Runtime.Cpu.UnreservedForVm
	resourcePoolInfo.AvailableMemory = moPool.Runtime.Memory.UnreservedForVm

	switch moPool.Parent.Type {
	case "ComputeResource":
		resourcePoolInfo.ParentType = ResourcePoolParentTypeHost
		resourcePoolInfo.ParentID = moPool.Parent.Value
	case "ClusterComputeResource":
		resourcePoolInfo.ParentType = ResourcePoolParentTypeCluster
		resourcePoolInfo.ParentID = moPool.Parent.Value
	case "VirtualApp":
		// 已经在helper层过滤掉了
		break
	case "ResourcePool":
		resourcePoolInfo.ParentType = ResourcePoolParentTypeResourcePool
		resourcePoolInfo.ParentID = moPool.Parent.Value
	}

	if moPool.Owner.Type == "ClusterComputeResource" {
		resourcePoolInfo.ClusterID = moPool.Owner.Value
	} else if moPool.Owner.Type == "ComputeResource" {
		resourcePoolInfo.HostID = moPool.Owner.Value
	} else {
		// 没有别的类型
	}
	return resourcePoolInfo
}

func (vc *VCenter) queryResourcePoolsFromCache(q protocol.ResourcePoolQuery) []protocol.ResourcePoolInfo {
	cache := vc.Cache.GetResourcePools()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var resourcePools []protocol.ResourcePoolInfo
		for _, resourcePool := range cache {
			if utils.SliceContain(q.IDs, resourcePool.ID) {
				resourcePools = append(resourcePools, resourcePool)
			}
		}
		return resourcePools
	} else if q.ClusterID != "" || q.HostID != "" {
		var resourcePools []protocol.ResourcePoolInfo
		for _, r := range cache {
			if q.ClusterID == r.ClusterID || q.HostID == r.HostID {
				resourcePools = append(resourcePools, r)
			}
		}
		return resourcePools
	} else if q.DatacenterID != "" {
		var resourcePools []protocol.ResourcePoolInfo
		for _, r := range cache {
			if q.DatacenterID == r.DatacenterID {
				resourcePools = append(resourcePools, r)
			}
		}
		return resourcePools
	} else {
		return cache
	}
}
