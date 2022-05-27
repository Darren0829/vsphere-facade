package cache

import (
	"vsphere_api/vsphere/protocol"
)

const StoragePolicyCacheKey = "StoragePolicy"

func (c VCCache) CacheStoragePolicies(v []protocol.StoragePolicyInfo) {
	c.Set(StoragePolicyCacheKey, v)
}

func (c VCCache) GetStoragePolicy(ID string) *protocol.StoragePolicyInfo {
	ResourcePoos := c.GetStoragePolicies()
	if ResourcePoos != nil {
		for _, c := range ResourcePoos {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetStoragePolicies() []protocol.StoragePolicyInfo {
	v, b := c.Get(StoragePolicyCacheKey)
	if !b {
		return nil
	}
	return v.([]protocol.StoragePolicyInfo)
}
