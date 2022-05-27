package cache

import (
	"vsphere_api/helper/datastore"
	"vsphere_api/vsphere/protocol"
)

func (c VCCache) CacheDatastores(v []protocol.DatastoreInfo) {
	c.Set(datastore.Type, v)
}

func (c VCCache) GetDatastore(ID string) *protocol.DatastoreInfo {
	Datastores := c.GetDatastores()
	if Datastores != nil {
		for _, c := range Datastores {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetDatastores() []protocol.DatastoreInfo {
	v, b := c.Get(datastore.Type)
	if !b {
		return nil
	}
	return v.([]protocol.DatastoreInfo)
}
