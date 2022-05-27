package vsphere

import (
	"context"
	"github.com/vmware/govmomi/pbm/types"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
	"vsphere_api/helper/spbm"
	"vsphere_api/helper/vsphere"
	"vsphere_api/vsphere/protocol"
)

func (vc *VCenter) QueryStoragePolicies(q protocol.StoragePolicyQuery) []protocol.StoragePolicyInfo {
	cache := vc.Cache.GetStoragePolicies()
	if cache != nil {
		logging.L().Debug("本次查询使用了缓存")
		return cache
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	pc := vsphere.GetPbmClient(vc.Api, ctx)
	if pc == nil {
		return nil
	}

	policies := spbm.GetPolicies(pc)
	if policies == nil {
		return nil
	}
	var policyInfos []protocol.StoragePolicyInfo
	for _, p := range *policies {
		policyInfo := buildStoragePolicyInfo(p)
		policyInfos = append(policyInfos, policyInfo)
	}

	vc.Cache.CacheStoragePolicies(policyInfos)
	return policyInfos
}

func buildStoragePolicyInfo(p types.PbmProfile) protocol.StoragePolicyInfo {
	var storagePolicyInfo protocol.StoragePolicyInfo
	storagePolicyInfo.ID = p.ProfileId.UniqueId
	storagePolicyInfo.Name = p.Name
	storagePolicyInfo.Description = p.Description
	return storagePolicyInfo
}
