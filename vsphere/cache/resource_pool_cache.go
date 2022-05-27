package cache

import (
	"vsphere-facade/helper/resourcepool"
	"vsphere-facade/vsphere/protocol"
)

func (c VCCache) CacheResourcePools(v []protocol.ResourcePoolInfo) {
	k := key(c.VCID, resourcepool.Type)
	c.Set(k, v)
}

func (c VCCache) GetResourcePool(ID string) *protocol.ResourcePoolInfo {
	ResourcePoos := c.GetResourcePools()
	if ResourcePoos != nil {
		for _, c := range ResourcePoos {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetResourcePools() []protocol.ResourcePoolInfo {
	v, b := c.Get(key(c.VCID, resourcepool.Type))
	if !b {
		return nil
	}
	return v.([]protocol.ResourcePoolInfo)
}
