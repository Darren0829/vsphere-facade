package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere_api/app/logging"
	"vsphere_api/app/utils"
	"vsphere_api/helper/network"
	"vsphere_api/vsphere/protocol"
)

const (
	NetworkTypeStandard      = "STANDARD_PORTGROUP"
	NetworkTypeDistributed   = "DISTRIBUTED_PORTGROUP"
	NetworkTypeOpaqueNetwork = "OPAQUE_NETWORK"
)

func (vc *VCenter) QueryNetworks(q protocol.NetworkQuery) []protocol.NetworkInfo {
	networks := vc.queryNetworksFromCache(q)
	if networks != nil {
		logging.L().Debug("本次查询使用了缓存")
		return networks
	}

	if len(q.IDs) > 0 {
		return vc.getNetworksByIDs(q.IDs)
	} else if q.DatacenterID != "" {
		return vc.getNetworksByDatacenterID(q.DatacenterID)
	} else {
		return vc.getNetworks()
	}
}

func (vc *VCenter) getNetworksByIDs(IDs []string) []protocol.NetworkInfo {
	var networkInfos []protocol.NetworkInfo
	for _, ID := range IDs {
		moNetwork := network.GetMObject(vc.Api, ID)
		if moNetwork == nil {
			continue
		}
		networkInfo := vc.buildNetworkInfo(*moNetwork, "")
		networkInfos = append(networkInfos, networkInfo)
	}
	return networkInfos
}

func (vc *VCenter) getNetworksByDatacenterID(datacenterID string) []protocol.NetworkInfo {
	moNetworks := network.GetByDatacenterID(vc.Api, datacenterID)
	if moNetworks == nil {
		return nil
	}
	var networkInfos []protocol.NetworkInfo
	for _, moNetwork := range *moNetworks {
		networkInfo := vc.buildNetworkInfo(moNetwork, datacenterID)
		networkInfos = append(networkInfos, networkInfo)
	}
	return networkInfos
}

func (vc *VCenter) getNetworks() []protocol.NetworkInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var networkInfos []protocol.NetworkInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moNetworks := network.GetByDatacenterID(vc.Api, datacenterID)
		if moNetworks == nil {
			continue
		}
		for _, moNetwork := range *moNetworks {
			networkInfo := vc.buildNetworkInfo(moNetwork, datacenterID)
			networkInfos = append(networkInfos, networkInfo)
		}
	}

	vc.Cache.CacheNetworks(networkInfos)
	return networkInfos
}

func (vc *VCenter) buildNetworkInfo(moNetwork mo.Network, datacenterID string) protocol.NetworkInfo {
	var networkInfo protocol.NetworkInfo
	networkInfo.DatacenterID = datacenterID
	networkInfo.ID = moNetwork.Reference().Value
	networkInfo.Name = moNetwork.Name
	networkInfo.Accessible = moNetwork.Summary.GetNetworkSummary().Accessible
	switch moNetwork.Reference().Type {
	case "Network":
		networkInfo.Type = NetworkTypeStandard
	case "DistributedVirtualPortgroup":
		networkInfo.Type = NetworkTypeDistributed
	case "OpaqueNetwork":
		networkInfo.Type = NetworkTypeOpaqueNetwork
	}
	return networkInfo
}

func (vc *VCenter) queryNetworksFromCache(q protocol.NetworkQuery) []protocol.NetworkInfo {
	cache := vc.Cache.GetNetworks()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var networks []protocol.NetworkInfo
		for _, network := range cache {
			if utils.SliceContain(q.IDs, network.ID) {
				networks = append(networks, network)
			}
		}
		return networks
	} else if q.DatacenterID != "" {
		var networks []protocol.NetworkInfo
		for _, network := range cache {
			if network.DatacenterID == q.DatacenterID {
				networks = append(networks, network)
			}
		}
		return networks
	} else {
		return cache
	}
}
