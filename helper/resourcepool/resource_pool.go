package resourcepool

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
	"vsphere_api/helper/clustercomputerresource"
	"vsphere_api/helper/computerresource"
	"vsphere_api/helper/datacenter"
	"vsphere_api/helper/virtualapp"
)

const Type = "ResourcePool"

func GetObject(api *helper.API, ID string) *object.ResourcePool {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取资源池", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  "ResourcePool",
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取资源池时发生错误", ID), err)
		return nil
	}
	return or.(*object.ResourcePool)
}

func GetMObject(api *helper.API, ID string) *mo.ResourcePool {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取资源池", ID))
	oResourcePool := GetObject(api, ID)
	if oResourcePool == nil {
		return nil
	}
	return FindProps(oResourcePool, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.ResourcePool {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有资源池", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	moResourcePools, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有资源池时发生错误", datacenterID), err)
		return nil
	}
	return moResourcePools
}

func GetByClusterID(api *helper.API, clusterID string) *[]mo.ResourcePool {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有资源池", clusterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  clustercomputerresource.Type,
		Value: clusterID,
	}
	moResourcePools, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有资源池时发生错误", clusterID), err)
		return nil
	}
	return moResourcePools
}

func GetByHostID(api *helper.API, computerResourceID string) *[]mo.ResourcePool {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有资源池", computerResourceID))
	moDatacenter := types.ManagedObjectReference{
		Type:  computerresource.Type,
		Value: computerResourceID,
	}
	moResourcePools, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有资源池时发生错误", computerResourceID), err)
		return nil
	}
	return moResourcePools
}

func FindProps(oResourcePool *object.ResourcePool, props []string) *mo.ResourcePool {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()

	var moResourcePool mo.ResourcePool
	err := oResourcePool.Properties(ctx, oResourcePool.Reference(), props, &moResourcePool)
	if err == nil {
		return nil
	}
	return &moResourcePool
}

func retrieve(api *helper.API, mor types.ManagedObjectReference) (*[]mo.ResourcePool, error) {
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	v, err := m.CreateContainerView(vctx, mor, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建资源池查询视图[%s]时发生错误", mor.Value), err)
		return nil, err
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var allResourcePools []mo.ResourcePool
	err = v.Retrieve(vctx, []string{Type}, nil, &allResourcePools)
	if err != nil {
		return nil, err
	}

	var moResourcePools []mo.ResourcePool
	// 过滤掉virtualApp和旗下的资源池
	// 过滤掉parent不是资源池的，因为parent不是资源池说明该资源池是虚拟的
	for _, pool := range allResourcePools {
		if pool.Parent.Type == virtualapp.Type {
			logging.L().Debug(fmt.Sprintf("过滤掉Virtual App[%s]", pool.Name))
			continue
		}
		if pool.Reference().Type == virtualapp.Type {
			logging.L().Debug(fmt.Sprintf("过滤掉Virtual App下的资源池[%s]", pool.Name))
			continue
		}
		if pool.Reference().Type != Type {
			logging.L().Debug(fmt.Sprintf("过滤掉父级不是资源池的资源池[%s]", pool.Name))
			continue
		}
		moResourcePools = append(moResourcePools, pool)
	}
	return &moResourcePools, nil
}
