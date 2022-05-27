package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere_api/app/logging"
	"vsphere_api/app/utils"
	"vsphere_api/helper/clustercomputerresource"
	"vsphere_api/helper/computerresource"
	"vsphere_api/helper/hostsystem"
	"vsphere_api/vsphere/protocol"
)

func (vc *VCenter) QueryHosts(q protocol.HostQuery) []protocol.HostInfo {
	hosts := vc.queryHostsFromCache(q)
	if hosts != nil {
		logging.L().Debug("本次查询使用了缓存")
		return hosts
	}

	if len(q.IDs) > 0 {
		return vc.getHostsByIDs(q.IDs)
	} else if q.ClusterID != "" {
		return vc.getHostsByCusterID(q.ClusterID)
	} else if q.DatacenterID != "" {
		return vc.getHostsByDatacenterID(q.DatacenterID)
	} else {
		return vc.getHosts()
	}
}

func (vc *VCenter) GetHostOSFamilies(hostID string) []protocol.OSFamilyInfo {
	host := vc.Cache.GetHost(hostID)
	if host != nil {
		return vc.GetComputerResourceOSFamilies(host.ParentId)
	}

	moHost := hostsystem.GetMObject(vc.Api, hostID)
	if moHost == nil {
		return nil
	}
	return vc.GetComputerResourceOSFamilies(moHost.Parent.Value)
}

func (vc *VCenter) getHostsByIDs(IDs []string) []protocol.HostInfo {
	var hostInfos []protocol.HostInfo
	for _, ID := range IDs {
		moHost := hostsystem.GetMObject(vc.Api, ID)
		if moHost == nil {
			continue
		}
		hostInfo := vc.buildHostInfo(*moHost, "")
		hostInfos = append(hostInfos, hostInfo)
	}
	return hostInfos
}

func (vc *VCenter) getHostsByCusterID(clusterID string) []protocol.HostInfo {
	moHosts := hostsystem.GetByClusterID(vc.Api, clusterID)
	if moHosts == nil {
		return nil
	}
	var hostInfos []protocol.HostInfo
	for _, moHost := range *moHosts {
		hostInfo := vc.buildHostInfo(moHost, "")
		hostInfos = append(hostInfos, hostInfo)
	}
	return hostInfos
}

func (vc *VCenter) getHostsByDatacenterID(datacenterID string) []protocol.HostInfo {
	moHosts := hostsystem.GetByDatacenterID(vc.Api, datacenterID)
	if moHosts == nil {
		return nil
	}
	var hostInfos []protocol.HostInfo
	for _, moHost := range *moHosts {
		hostInfo := vc.buildHostInfo(moHost, datacenterID)
		hostInfos = append(hostInfos, hostInfo)
	}
	return hostInfos
}

func (vc *VCenter) getHosts() []protocol.HostInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var hostInfos []protocol.HostInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moHosts := hostsystem.GetByDatacenterID(vc.Api, datacenterID)
		if moHosts == nil {
			continue
		}
		for _, moHost := range *moHosts {
			hostInfo := vc.buildHostInfo(moHost, datacenterID)
			hostInfos = append(hostInfos, hostInfo)
		}
	}

	vc.Cache.CacheHosts(hostInfos)
	return hostInfos
}

func (vc *VCenter) buildHostInfo(moHost mo.HostSystem, datacenterID string) protocol.HostInfo {
	var hostInfo protocol.HostInfo
	hostInfo.ID = moHost.Reference().Value
	hostInfo.Name = moHost.Name
	hostInfo.DatacenterID = datacenterID
	hostInfo.ParentId = moHost.Parent.Value

	var datastores []string
	for _, datastore := range moHost.Datastore {
		datastores = append(datastores, datastore.Value)
	}
	hostInfo.Datastores = datastores

	var networks []string
	for _, network := range moHost.Network {
		networks = append(networks, network.Value)
	}
	hostInfo.Networks = networks

	if moHost.Parent.Type == clustercomputerresource.Type {
		hostInfo.ClusterID = moHost.Parent.Value
	}

	if moHost.Parent.Type == computerresource.Type {
		hostInfo.ResourcePoolId = moHost.Parent.Value
	}
	return hostInfo
}

func (vc *VCenter) queryHostsFromCache(q protocol.HostQuery) []protocol.HostInfo {
	cache := vc.Cache.GetHosts()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var hosts []protocol.HostInfo
		for _, host := range cache {
			if utils.SliceContain(q.IDs, host.ID) {
				hosts = append(hosts, host)
			}
		}
		return hosts
	} else if q.ClusterID != "" {
		var hosts []protocol.HostInfo
		for _, host := range cache {
			if host.ClusterID == q.ClusterID {
				hosts = append(hosts, host)
			}
		}
		return hosts
	} else if q.DatacenterID != "" {
		var hosts []protocol.HostInfo
		for _, host := range cache {
			if host.DatacenterID == q.DatacenterID {
				hosts = append(hosts, host)
			}
		}
		return hosts
	} else {
		return cache
	}
}
