package folder

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

const Type = "Folder"

func GetObject(api *helper.API, ID string) *object.Folder {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取文件夹", ID))
	if ID == "" {
		return nil
	}

	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取文件夹时,发生错误", ID), err)
		return nil
	}

	return or.(*object.Folder)
}

func GetMObject(api *helper.API, ID string) *mo.Folder {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取文件夹", ID))
	oFolder := GetObject(api, ID)
	if oFolder == nil {
		return nil
	}
	return FindProps(oFolder, nil)
}

func GetByDatacenterID(api *helper.API, datacenterID string) *[]mo.Folder {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有文件夹", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}
	folders, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有文件夹时发生错误", datacenterID), err)
		return nil
	}
	return folders
}

func FindProps(oFolder *object.Folder, props []string) *mo.Folder {
	logging.L().Debug(fmt.Sprintf("获取文件夹[%s]属性", oFolder.Name()))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moFolder mo.Folder
	if err := oFolder.Properties(ctx, oFolder.Reference(), props, &moFolder); err != nil {
		return nil
	}
	return &moFolder
}

func GetVMRootFolder(api *helper.API, datacenterID string) *object.Folder {
	logging.L().Debug(fmt.Sprintf("获取数据中心[%s]下的虚拟机根文件夹", datacenterID))
	moDC := datacenter.GetMObject(api, datacenterID)
	if moDC == nil {
		return nil
	}

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: moDC.VmFolder.Value,
	}

	finder := find.NewFinder(api.Client.Client, false)
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, _ := finder.ObjectReference(ctx, ref)
	return or.(*object.Folder)
}

func GetVMFoldersByDatacenterID(api *helper.API, datacenterID string) *[]mo.Folder {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有虚拟机文件夹", datacenterID))
	moDC := datacenter.GetMObject(api, datacenterID)
	if moDC == nil {
		return nil
	}

	moVMRootFolder := types.ManagedObjectReference{
		Type:  Type,
		Value: moDC.VmFolder.Value,
	}
	folders, err := retrieve(api, moVMRootFolder)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有虚拟机文件夹时发生错误", datacenterID), err)
		return nil
	}
	return folders
}

func GetByFolderID(api *helper.API, folderID string) *[]mo.Folder {
	logging.L().Debug(fmt.Sprintf("查询文件夹[%s]下的所有文件夹", folderID))
	moDatacenter := types.ManagedObjectReference{
		Type:  Type,
		Value: folderID,
	}
	folders, err := retrieve(api, moDatacenter)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询文件夹[%s]下的所有文件夹时发生错误", folderID), err)
		return nil
	}
	return folders
}

func retrieve(api *helper.API, mor types.ManagedObjectReference) (*[]mo.Folder, error) {
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

	var moResourcePools []mo.Folder
	err = v.Retrieve(vctx, []string{Type}, nil, &moResourcePools)
	if err != nil {
		return nil, err
	}
	return &moResourcePools, nil
}
