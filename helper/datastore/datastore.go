package datastore

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
	"vsphere_api/helper/datacenter"
)

const Type = "Datastore"

type FormatMapping struct {
	ThinProvisioned *bool
	EagerlyScrub    *bool
}

func GetObject(api *helper.API, ID string) *object.Datastore {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取存储", ID))
	finder := find.NewFinder(api.Client.Client, false)
	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取存储时发生错误", ID), err)
		return nil
	}
	return or.(*object.Datastore)
}

func GetMObject(api *helper.API, ID string) *mo.Datastore {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取计算资源", ID))
	oDatastore := GetObject(api, ID)
	if oDatastore == nil {
		return nil
	}
	return FindProps(oDatastore, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.Datastore {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的存储", datacenterID))
	scope := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	oDatastores, err := retrieve(api, scope)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下所有的存储时发生错误", scope.Value), err)
		return nil
	}
	return oDatastores
}

func FindProps(oDatastore *object.Datastore, props []string) *mo.Datastore {
	logging.L().Debug(fmt.Sprintf("获取存储[%s]属性", oDatastore.Name()))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moDatastore mo.Datastore
	if err := oDatastore.Properties(ctx, oDatastore.Reference(), props, &moDatastore); err != nil {
		return nil
	}
	return &moDatastore
}

func retrieve(api *helper.API, scope types.ManagedObjectReference) (*[]mo.Datastore, error) {
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	v, err := m.CreateContainerView(vctx, scope, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建存储查询视图[%s]时发生错误", scope.Value), err)
		return nil, err
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var moDatastores []mo.Datastore
	err = v.Retrieve(vctx, []string{Type}, nil, &moDatastores)
	if err != nil {
		return nil, err
	}
	return &moDatastores, nil
}
