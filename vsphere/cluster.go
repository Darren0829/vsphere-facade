package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/helper/clustercomputerresource"
	"vsphere-facade/vsphere/protocol"
)

func (vc *VCenter) QueryClusters(q protocol.ClusterQuery) []protocol.ClusterInfo {
	clusters := vc.queryClustersFromCache(q)
	if clusters != nil {
		logging.L().Debug("本次查询使用了缓存")
		return clusters
	}

	if len(q.IDs) > 0 {
		return vc.getClustersByIDs(q.IDs)
	} else if q.DatacenterID != "" {
		return vc.getClustersByDatacenterID(q.DatacenterID)
	} else {
		return vc.getClusters()
	}
}

func (vc *VCenter) getClustersByIDs(ids []string) []protocol.ClusterInfo {
	var clusterInfos []protocol.ClusterInfo
	for _, ID := range ids {
		moCluster := clustercomputerresource.GetMObject(vc.Api, ID)
		if moCluster == nil {
			continue
		}
		clusterInfo := vc.buildClusterInfo(*moCluster, "")
		clusterInfos = append(clusterInfos, clusterInfo)
	}
	return clusterInfos
}

func (vc *VCenter) getClustersByDatacenterID(datacenterID string) []protocol.ClusterInfo {
	moClusters := clustercomputerresource.GetByDatacenterID(vc.Api, datacenterID)
	if moClusters == nil {
		return nil
	}
	var clusterInfos []protocol.ClusterInfo
	for _, moCluster := range *moClusters {
		var clusterInfo protocol.ClusterInfo
		clusterInfo = vc.buildClusterInfo(moCluster, datacenterID)
		clusterInfos = append(clusterInfos, clusterInfo)
	}
	return clusterInfos
}

func (vc *VCenter) getClusters() []protocol.ClusterInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var clusterInfos []protocol.ClusterInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moClusters := clustercomputerresource.GetByDatacenterID(vc.Api, datacenterID)
		if moClusters == nil {
			continue
		}

		for _, oCluster := range *moClusters {
			clusterInfo := vc.buildClusterInfo(oCluster, datacenterID)
			clusterInfos = append(clusterInfos, clusterInfo)
		}
	}

	vc.Cache.CacheClusters(clusterInfos)
	return clusterInfos
}

func (vc *VCenter) buildClusterInfo(moCluster mo.ClusterComputeResource, datacenterID string) protocol.ClusterInfo {
	var clusterInfo protocol.ClusterInfo
	clusterInfo.DatacenterID = datacenterID
	clusterInfo.ID = moCluster.Reference().Value
	clusterInfo.Name = moCluster.Name
	clusterInfo.DrsEnabled = *moCluster.Configuration.DrsConfig.Enabled
	clusterInfo.ResourcePoolId = moCluster.ResourcePool.Value
	return clusterInfo
}

func (vc *VCenter) queryClustersFromCache(q protocol.ClusterQuery) []protocol.ClusterInfo {
	cache := vc.Cache.GetClusters()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var clusters []protocol.ClusterInfo
		for _, cluster := range cache {
			if utils.SliceContain(q.IDs, cluster.ID) {
				clusters = append(clusters, cluster)
			}
		}
		return clusters
	} else if q.DatacenterID != "" {
		var clusters []protocol.ClusterInfo
		for _, cluster := range cache {
			if cluster.DatacenterID == q.DatacenterID {
				clusters = append(clusters, cluster)
			}
		}
		return clusters
	} else {
		return cache
	}
}
