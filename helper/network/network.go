package network

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"vsphere-facade/app/logging"
	"vsphere-facade/helper"
	"vsphere-facade/helper/datacenter"
)

const Type = "Network"

func GetObject(api *helper.API, ID string) *object.Network {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取网络", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取网络时,发生错误", ID), err)
		return nil
	}

	return or.(*object.Network)
}

func GetMObject(api *helper.API, ID string) *mo.Network {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取网络", ID))
	oNetwork := GetObject(api, ID)
	if oNetwork == nil {
		return nil
	}
	return FindProps(oNetwork, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.Network {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有网络", datacenterID))
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	v, err := m.CreateContainerView(vctx, moDatacenter, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建数据中心[%s]网络查询视图时发生错误", datacenterID), err)
		return nil
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var networks []mo.Network
	err = v.Retrieve(vctx, []string{Type}, nil, &networks)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有网络时发生错误", datacenterID), err)
		return nil
	}
	return &networks
}

func FindProps(oNetwork *object.Network, props []string) *mo.Network {
	logging.L().Debug(fmt.Sprintf("获取网络[%s]属性", oNetwork.Name()))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moNetwork mo.Network
	if err := oNetwork.Properties(ctx, oNetwork.Reference(), props, &moNetwork); err != nil {
		return nil
	}
	return &moNetwork
}
