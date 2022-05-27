package hostsystem

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
	"vsphere-facade/helper/clustercomputerresource"
	"vsphere-facade/helper/computerresource"
	"vsphere-facade/helper/datacenter"
)

const Type = "HostSystem"

func GetObject(api *helper.API, ID string) *object.HostSystem {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取主机", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取主机时发生错误", ID), err)
		return nil
	}
	return or.(*object.HostSystem)
}

func GetMObject(api *helper.API, ID string) *mo.HostSystem {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取网络", ID))
	oHostSystem := GetObject(api, ID)
	if oHostSystem == nil {
		return nil
	}
	return FindProps(oHostSystem, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.HostSystem {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有主机", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	moHostSystems, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有主机时发生错误", datacenterID), err)
		return nil
	}
	return moHostSystems
}

func GetByClusterID(api *helper.API, clusterID string) *[]mo.HostSystem {
	logging.L().Debug(fmt.Sprintf("查询群集[%s]下的所有主机", clusterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  clustercomputerresource.Type,
		Value: clusterID,
	}
	moHostSystems, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询群集[%s]下的所有主机时发生错误", clusterID), err)
		return nil
	}
	return moHostSystems
}

func FindProps(oHostSystem *object.HostSystem, props []string) *mo.HostSystem {
	logging.L().Debug(fmt.Sprintf("获取主机[%s]属性", oHostSystem.Name()))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moHostSystem mo.HostSystem
	if err := oHostSystem.Properties(ctx, oHostSystem.Reference(), props, &moHostSystem); err != nil {
		return nil
	}
	return &moHostSystem
}

func GetParentPool(api *helper.API, ID string) *types.ManagedObjectReference {
	oHost := GetObject(api, ID)
	if oHost == nil {
		return nil
	}

	props := FindProps(oHost, []string{"parent"})
	if props == nil {
		return nil
	}

	if props.Parent.Type == clustercomputerresource.Type {
		moCluster := clustercomputerresource.GetMObject(api, props.Parent.Value)
		return moCluster.ResourcePool
	} else {
		moComputer := computerresource.GetMObject(api, props.Parent.Value)
		return moComputer.ResourcePool
	}
}

func GetCluster(api *helper.API, ID string) *mo.ClusterComputeResource {
	oHost := GetObject(api, ID)
	if oHost == nil {
		return nil
	}

	props := FindProps(oHost, []string{"parent"})
	if props == nil {
		return nil
	}

	if props.Parent.Type == clustercomputerresource.Type {
		moCluster := clustercomputerresource.GetMObject(api, props.Parent.Value)
		return moCluster
	}
	return nil
}

func retrieve(api *helper.API, mor types.ManagedObjectReference) (*[]mo.HostSystem, error) {
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	v, err := m.CreateContainerView(vctx, mor, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建主机查询视图[%s]时发生错误", mor.Value), err)
		return nil, err
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var moResourcePools []mo.HostSystem
	err = v.Retrieve(vctx, []string{Type}, nil, &moResourcePools)
	if err != nil {
		return nil, err
	}
	return &moResourcePools, nil
}
