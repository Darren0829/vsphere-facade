package cache

import (
	"vsphere-facade/vsphere/protocol"
)

const OSFamilyCacheKey = "OSFamily"

func (c VCCache) CacheOSFamilies(v []protocol.OSFamilyInfo) {
	k := key(c.VCID, OSFamilyCacheKey)
	c.Set(k, v)
}

func (c VCCache) GetOSFamily(ID string) *protocol.OSFamilyInfo {
	families := c.GetOSFamilies()
	if families != nil {
		for _, f := range families {
			if f.ID == ID {
				return &f
			}
		}
	}
	return nil
}

func (c VCCache) GetOSFamilies() []protocol.OSFamilyInfo {
	v, b := c.Get(OSFamilyCacheKey)
	if !b {
		return nil
	}
	return v.([]protocol.OSFamilyInfo)
}
