package cache

import (
	"vsphere-facade/helper/datacenter"
	"vsphere-facade/vsphere/protocol"
)

func (c VCCache) CacheDatacenters(v []protocol.DatacenterInfo) {
	c.Set(datacenter.Type, v)
}

func (c VCCache) GetDatacenter(ID string) *protocol.DatacenterInfo {
	dcs := c.GetDatacenters()
	if dcs != nil {
		for _, dc := range dcs {
			if dc.ID == ID {
				return &dc
			}
		}
	}
	return nil
}

func (c VCCache) GetDatacenters() []protocol.DatacenterInfo {
	v, b := c.Get(datacenter.Type)
	if !b {
		return nil
	}
	return v.([]protocol.DatacenterInfo)
}
