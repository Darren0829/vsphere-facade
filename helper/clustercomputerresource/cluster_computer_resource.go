package clustercomputerresource

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

const Type = "ClusterComputeResource"

func GetObject(api *helper.API, ID string) *object.ClusterComputeResource {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取集群", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取集群时发生错误", ID), err)
		return nil
	}
	return or.(*object.ClusterComputeResource)
}

func GetMObject(api *helper.API, ID string) *mo.ClusterComputeResource {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取集群", ID))
	oCluster := GetObject(api, ID)
	if oCluster == nil {
		return nil
	}
	return FindProps(oCluster, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.ClusterComputeResource {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的集群资源", datacenterID))
	scope := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	oClusters, err := retrieve(api, scope)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有集群时发生错误", datacenterID), err)
		return nil
	}
	return oClusters
}

func FindProps(oCluster *object.ClusterComputeResource, props []string) *mo.ClusterComputeResource {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moCluster mo.ClusterComputeResource
	if err := oCluster.Properties(ctx, oCluster.Reference(), props, &moCluster); err != nil {
		logging.L().Error(fmt.Sprintf("查询集群[%s]属性[%s]时发生错误", oCluster.Name(), props), err)
		return nil
	}
	return &moCluster
}

func retrieve(api *helper.API, scope types.ManagedObjectReference) (*[]mo.ClusterComputeResource, error) {
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	v, err := m.CreateContainerView(vctx, scope, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建集群查询视图[%s]时发生错误", scope.Value), err)
		return nil, err
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var moClusterComputeResources []mo.ClusterComputeResource
	err = v.Retrieve(vctx, []string{Type}, nil, &moClusterComputeResources)
	if err != nil {
		return nil, err
	}
	return &moClusterComputeResources, nil
}
