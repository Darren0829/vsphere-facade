package cache

import (
	"vsphere-facade/helper/hostsystem"
	"vsphere-facade/vsphere/protocol"
)

func (c VCCache) CacheHosts(v []protocol.HostInfo) {
	c.Set(hostsystem.Type, v)
}

func (c VCCache) GetHost(ID string) *protocol.HostInfo {
	Hosts := c.GetHosts()
	if Hosts != nil {
		for _, c := range Hosts {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetHosts() []protocol.HostInfo {
	v, b := c.Get(hostsystem.Type)
	if !b {
		return nil
	}
	return v.([]protocol.HostInfo)
}
