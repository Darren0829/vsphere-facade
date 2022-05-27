package computerresource

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
	"vsphere-facade/helper/envbrowser"
)

const Type = "ComputeResource"

func GetObject(api *helper.API, ID string) *object.ComputeResource {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取计算资源", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取计算资源时发生错误", ID), err)
		return nil
	}
	return or.(*object.ComputeResource)
}

func GetMObject(api *helper.API, ID string) *mo.ComputeResource {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取计算资源", ID))
	oCluster := GetObject(api, ID)
	if oCluster == nil {
		return nil
	}
	return FindProps(oCluster, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) []mo.ComputeResource {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的计算资源", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	moComputeResources, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有计算资源时发生错误", datacenterID), err)
		return nil
	}
	return moComputeResources
}

func FindProps(oCluster *object.ComputeResource, props []string) *mo.ComputeResource {
	logging.L().Debug(fmt.Sprintf("获取计算资源[%s(%s)]属性: %s", oCluster.Name(), oCluster.Reference().Value, props))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moCluster mo.ComputeResource
	if err := oCluster.Properties(ctx, oCluster.Reference(), props, &moCluster); err != nil {
		logging.L().Error(fmt.Sprintf("获取计算资源[%s]属性[%s]时发生错误", oCluster.Name(), props), err)
		return nil
	}
	return &moCluster
}

func GetOSFamilies(api *helper.API, id string) []types.GuestOsDescriptor {
	envBrowser := getEnvBrowser(api, id)
	if envBrowser == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	return envBrowser.GetOSFamilies(ctx)
}

func GetOSFamily(api *helper.API, id string, guestID string) *types.GuestOsDescriptor {
	families := GetOSFamilies(api, id)
	for _, f := range families {
		if f.Id == guestID {
			return &f
		}
	}
	return nil
}

func retrieve(api *helper.API, scope types.ManagedObjectReference) ([]mo.ComputeResource, error) {
	m := view.NewManager(api.Client.Client)
	vctx, vcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer vcancel()

	v, err := m.CreateContainerView(vctx, scope, []string{Type}, true)
	if err != nil {
		logging.L().Error(fmt.Sprintf("创建计算资源查询视图[%s]时发生错误", scope.Value), err)
		return nil, err
	}

	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer dcancel()
		v.Destroy(dctx)
	}()

	var moComputeResources []mo.ComputeResource
	err = v.Retrieve(vctx, []string{Type}, nil, &moComputeResources)
	if err != nil {
		return nil, err
	}
	return moComputeResources, nil
}

func getEnvBrowser(api *helper.API, ID string) *envbrowser.EnvironmentBrowser {
	logging.L().Debug(fmt.Sprintf("获取计算资源[%s]的environmentBrowser", ID))
	oCluster := GetObject(api, ID)
	if oCluster == nil {
		return nil
	}
	props := FindProps(oCluster, []string{"environmentBrowser"})
	if props == nil || props.EnvironmentBrowser == nil {
		logging.L().Debug(fmt.Sprintf("计算资源[%s]的environmentBrowser为空", ID))
		return nil
	}
	return envbrowser.New(api.Client.Client, *props.EnvironmentBrowser)
}
