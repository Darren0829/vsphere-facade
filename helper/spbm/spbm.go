package spbm

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/pbm"
	"github.com/vmware/govmomi/pbm/types"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
)

func GetPolicies(c *pbm.Client) *[]types.PbmProfile {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()

	rtype := types.PbmProfileResourceType{
		ResourceType: string(types.PbmProfileResourceTypeEnumSTORAGE),
	}
	category := types.PbmProfileCategoryEnumREQUIREMENT
	profileIds, err := c.QueryProfile(ctx, rtype, string(category))
	if err != nil {
		logging.L().Error("查询存储策略时发生错误", err)
		return nil
	}

	profiles, err := c.RetrieveContent(ctx, profileIds)
	if err != nil {
		logging.L().Error("查询存储策略时发生错误", err)
		return nil
	}

	var policies []types.PbmProfile
	for _, p := range profiles {
		policies = append(policies, *p.GetPbmProfile())
	}
	return &policies
}

func GetPolicy(c *pbm.Client, id string) *types.PbmProfile {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()

	profileId := []types.PbmProfileId{
		{
			UniqueId: id,
		},
	}
	policies, err := c.RetrieveContent(ctx, profileId)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]查询存储策略时发生错误", id), err)
	}
	return policies[0].GetPbmProfile()
}

//go get -u github.com/swaggo/swag/cmd/swag
//go get -u github.com/swaggo/gin-swagger
//go get -u github.com/swaggo/files
//go get -u github.com/alecthomas/template
