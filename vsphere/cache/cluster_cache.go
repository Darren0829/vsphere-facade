package cache

import (
	"vsphere-facade/helper/computerresource"
	"vsphere-facade/vsphere/protocol"
)

func (c VCCache) CacheClusters(v []protocol.ClusterInfo) {
	c.Set(computerresource.Type, v)
}

func (c VCCache) GetCluster(ID string) *protocol.ClusterInfo {
	clusters := c.GetClusters()
	if clusters != nil {
		for _, c := range clusters {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetClusters() []protocol.ClusterInfo {
	v, b := c.Get(computerresource.Type)
	if !b {
		return nil
	}
	return v.([]protocol.ClusterInfo)
}
