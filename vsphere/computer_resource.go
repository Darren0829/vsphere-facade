package vsphere

import (
	"github.com/vmware/govmomi/vim25/types"
	"vsphere-facade/app/utils"
	"vsphere-facade/helper/computerresource"
	"vsphere-facade/vsphere/protocol"
)

func (vc *VCenter) GetComputerResourceOSFamilies(computerResourceID string) []protocol.OSFamilyInfo {
	families := computerresource.GetOSFamilies(vc.Api, computerResourceID)
	if families == nil {
		return nil
	}
	var familyInfos []protocol.OSFamilyInfo
	for _, f := range families {
		var familyInfo = vc.buildOSInfo(f)
		familyInfos = append(familyInfos, familyInfo)
	}
	return familyInfos
}

func (vc *VCenter) getOSFamilies() []protocol.OSFamilyInfo {
	families := vc.Cache.GetOSFamilies()
	if families != nil {
		return families
	}

	var existIDs []string
	var osFamilyInfos []protocol.OSFamilyInfo
	dcs := vc.getDatacenters()
	if dcs != nil {
		for _, dc := range dcs {
			list := computerresource.GetByDatacenterID(vc.Api, dc.ID)
			if list == nil {
				continue
			}
			for _, cr := range list {
				families := computerresource.GetOSFamilies(vc.Api, cr.Reference().Value)
				if families == nil {
					continue
				}
				for _, f := range families {
					if utils.SliceContain(existIDs, f.Id) {
						continue
					}
					var familyInfo = vc.buildOSInfo(f)
					osFamilyInfos = append(osFamilyInfos, familyInfo)
				}
			}
		}
	}
	vc.Cache.CacheOSFamilies(osFamilyInfos)
	return osFamilyInfos
}

func (vc *VCenter) buildOSInfo(o types.GuestOsDescriptor) protocol.OSFamilyInfo {
	var familyInfo = protocol.OSFamilyInfo{}
	familyInfo.ID = o.Id
	familyInfo.Name = o.FullName
	familyInfo.Family = o.Family
	familyInfo.SupportedMinMen = o.SupportedMinMemMB
	familyInfo.SupportedMaxMen = o.SupportedMaxMemMB
	familyInfo.SupportedCPUs = o.SupportedMaxCPUs
	return familyInfo
}
