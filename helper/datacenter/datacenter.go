package datacenter

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
)

const Type = "Datacenter"

func GetObject(api *helper.API, ID string) *object.Datacenter {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取数据中心", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取数据中心时发生错误", ID), err)
		return nil
	}
	return or.(*object.Datacenter)
}

func GetMObject(api *helper.API, ID string) *mo.Datacenter {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取数据中心", ID))
	oDatacenter := GetObject(api, ID)
	if oDatacenter == nil {
		return nil
	}
	return FindProps(oDatacenter, nil)
}

func GetAll(api *helper.API) []*object.Datacenter {
	logging.L().Debug("获取所有的数据中心")
	finder := find.NewFinder(api.Client.Client, false)
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	dcs, err := finder.DatacenterList(ctx, "*")
	if err != nil {
		logging.L().Error("查询所有数据中心时发生错误", err)
		return nil
	}
	return dcs
}

func FindProps(oDatacenter *object.Datacenter, props []string) *mo.Datacenter {
	logging.L().Debug(fmt.Sprintf("获取数据中心[%s]属性", oDatacenter.Name()))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moDatacenter mo.Datacenter
	if err := oDatacenter.Properties(ctx, oDatacenter.Reference(), props, &moDatacenter); err != nil {
		return nil
	}
	return &moDatacenter
}
