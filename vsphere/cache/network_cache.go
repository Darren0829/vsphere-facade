package cache

import (
	"vsphere-facade/helper/network"
	"vsphere-facade/vsphere/protocol"
)

func (c VCCache) CacheNetworks(v []protocol.NetworkInfo) {
	c.Set(network.Type, v)
}

func (c VCCache) GetNetwork(ID string) *protocol.NetworkInfo {
	Networks := c.GetNetworks()
	if Networks != nil {
		for _, c := range Networks {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetNetworks() []protocol.NetworkInfo {
	v, b := c.Get(network.Type)
	if !b {
		return nil
	}
	return v.([]protocol.NetworkInfo)
}
