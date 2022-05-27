package virtualmachineclone

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"time"
	"vsphere-facade/app/logging"
	"vsphere-facade/helper"
	"vsphere-facade/helper/clustercomputerresource"
	"vsphere-facade/helper/datastore"
	"vsphere-facade/helper/disk"
	"vsphere-facade/helper/folder"
	"vsphere-facade/helper/hostsystem"
	"vsphere-facade/helper/resourcepool"
	"vsphere-facade/helper/virtualmachine"
)

type CloneParameter struct {
	ID       string                        `json:"id"`
	Name     string                        `json:"name"`
	Location LocationParameter             `json:"location"`
	Disks    *[]SelfContainedDiskParameter `json:"disks"`
}

type LocationParameter struct {
	DatacenterID   string  `json:"datacenterId"`
	FolderID       *string `json:"folderId,omitempty"`
	ClusterID      *string `json:"clusterId,omitempty"`
	HostId         *string `json:"hostId,omitempty"`
	ResourcePoolID *string `json:"resourcePoolId,omitempty"`
	DatastoreID    *string `json:"datastoreId"`
}

type SelfContainedDiskParameter struct {
	Key             int32   `json:"key"`
	Mode            *string `json:"mode"`
	Format          *string `json:"format"`
	DatastoreID     *string `json:"datastoreId"`
	StoragePolicyID *string `json:"storagePolicyId"`
}

func Clone(api *helper.API, p CloneParameter, timeout int32) (*object.VirtualMachine, error) {
	logging.L().Debug(fmt.Sprintf("从虚拟机/模版[%s]克隆虚拟机，超时时间为[%dm]", p.ID, timeout))
	oVM := virtualmachine.GetObject(api, p.ID)
	if oVM == nil {
		return nil, fmt.Errorf("克隆的虚拟机/模版[%s]无法找到", p.ID)
	}

	var oFolder *object.Folder
	if p.Location.FolderID != nil {
		oFolder = folder.GetObject(api, *p.Location.FolderID)
	}
	if oFolder == nil {
		oFolder = folder.GetVMRootFolder(api, p.Location.DatacenterID)
	}

	var cloneSpec = types.VirtualMachineCloneSpec{}
	var location = types.VirtualMachineRelocateSpec{}
	// 存储
	var err error
	err = parseLocationDatastore(api, p, &location)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	// 文件夹
	err = parseLocationFolder(api, p, &location)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	// 主机
	err = parseLocationHost(api, p, &location)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	// 资源池
	err = parseLocationPool(api, p, &location)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	// 模版磁盘
	props := virtualmachine.FindProps(oVM, "config")
	devices := object.VirtualDeviceList(props.Config.Hardware.Device)
	err = parseCloneDisk(api, p, devices, &location)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	cloneSpec.Location = location

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
	defer cancel()
	oTask, err := oVM.Clone(ctx, oFolder, p.Name, cloneSpec)
	if err != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, err)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]超时[>%dm]", p.ID, timeout)
	}
	taskInfo, err := oTask.WaitForResult(ctx, nil)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("克隆虚拟机/模版[%s]任务等待超时[>%dm]", p.ID, timeout)
		}
	}
	if taskInfo.Error != nil {
		return nil, fmt.Errorf("克隆虚拟机/模版[%s]失败: %s", p.ID, taskInfo.Error.LocalizedMessage)
	}
	ID := taskInfo.Result.(types.ManagedObjectReference).Value
	logging.L().Debug(fmt.Sprintf("克隆虚拟机/模版[%s]完成，新的虚拟机ID: %s", p.ID, ID))
	return virtualmachine.GetObject(api, ID), nil
}

func parseCloneDisk(api *helper.API, p CloneParameter, devices []types.BaseVirtualDevice, location *types.VirtualMachineRelocateSpec) error {
	if p.Disks == nil {
		return nil
	}

	var dpm = make(map[int32]*SelfContainedDiskParameter)
	for _, diskParameter := range *p.Disks {
		dpm[diskParameter.Key] = &diskParameter
	}

	virtualDeviceList := object.VirtualDeviceList(devices)
	var diskLocators []types.VirtualMachineRelocateSpecDiskLocator
	diskDevices := virtualDeviceList.SelectByType((*types.VirtualDisk)(nil))
	for _, diskDevice := range diskDevices {
		d := diskDevice.GetVirtualDevice()
		vd := diskDevice.(*types.VirtualDisk)
		backing := vd.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
		var diskLocator types.VirtualMachineRelocateSpecDiskLocator
		diskLocator.DiskId = d.Key
		dp := dpm[d.Key]
		if dp != nil {
			format := disk.FormatMapping[*dp.Format]
			if format != nil {
				backing.EagerlyScrub = format.EagerlyScrub
				backing.ThinProvisioned = format.ThinProvisioned
			}
			if dp.Mode != nil {
				backing.DiskMode = *dp.Mode
			}
			if dp.StoragePolicyID != nil {
				diskLocator.Profile = []types.BaseVirtualMachineProfileSpec{
					&types.VirtualMachineDefinedProfileSpec{
						ProfileId: *dp.StoragePolicyID,
					},
				}
			}
			if dp.DatastoreID != nil {
				oDatastore := datastore.GetObject(api, *dp.DatastoreID)
				if oDatastore == nil {
					return fmt.Errorf("存储[%s]不存在", *dp.DatastoreID)
				}
				diskLocator.Datastore = oDatastore.Reference()
			}
		}
		if diskLocator.Datastore.Type == "" {
			if location.Datastore != nil {
				diskLocator.Datastore = *location.Datastore
				logging.L().Debugf("硬盘未设置存储ID参数，使用location参数中设置的存储[%s]", location.Datastore.Value)
			} else {
				diskLocator.Datastore = *backing.Datastore
				logging.L().Debugf("硬盘未设置存储ID参数，使用模版硬盘所在存储[%s]", backing.Datastore.Value)
			}
		}
		diskLocator.DiskBackingInfo = backing
		diskLocators = append(diskLocators, diskLocator)
	}
	location.Disk = diskLocators
	return nil
}

//parseLocationDatastore
//The datastore where the virtual machine should be located. If not specified, the current datastore is used.
func parseLocationDatastore(api *helper.API, p CloneParameter, location *types.VirtualMachineRelocateSpec) error {
	l := p.Location
	if l.DatastoreID == nil {
		logging.L().Debug("未设置location.datastore")
		return nil
	}
	oDatastore := datastore.GetObject(api, *l.DatastoreID)
	if oDatastore == nil {
		return fmt.Errorf("存储[%s]无法找到", *l.DatastoreID)
	}
	datastoreRef := oDatastore.Reference()
	location.Datastore = &datastoreRef
	return nil
}

//parseLocationFolder
//The folder where the virtual machine should be located. If not specified, the root VM folder of the destination datacenter will be used.
//Since vSphere API 6.0
func parseLocationFolder(api *helper.API, p CloneParameter, location *types.VirtualMachineRelocateSpec) error {
	l := p.Location
	if l.FolderID == nil {
		return nil
	}

	if api.Newer(6, 0, 0) {
		oFolder := folder.GetObject(api, *l.FolderID)
		if oFolder == nil {
			return fmt.Errorf("文件夹[%s]无法找到", *l.FolderID)
		}
		folderRef := oFolder.Reference()
		location.Folder = &folderRef
	} else {
		logging.L().Debug("版本小于6.0，跳过设置location.folder")
	}
	return nil
}

//parseLocationHost
//The target host for the virtual machine. If not specified,
//    if resource pool is not specified, current host is used.
//    if resource pool is specified, and the target pool represents a stand-alone host, the host is used.
//    if resource pool is specified, and the target pool represents a DRS-enabled cluster, a host selected by DRS is used.
//    if resource pool is specified and the target pool represents a cluster without DRS enabled, an InvalidArgument exception be thrown.
//    if the virtual machine is relocated to a different vCenter service, both the destination host has to be specified and cannot be unset.
func parseLocationHost(api *helper.API, p CloneParameter, location *types.VirtualMachineRelocateSpec) error {
	l := p.Location
	if l.HostId == nil {
		return nil
	}

	if api.Newer(6, 0, 0) {
		oHost := hostsystem.GetObject(api, *l.HostId)
		if oHost == nil {
			return fmt.Errorf("主机[%s]无法找到", *l.HostId)
		}
		hostRef := oHost.Reference()
		location.Host = &hostRef
	} else {
		logging.L().Debug("版本小于6.0，跳过设置location.folder")
	}
	return nil
}

//parseLocationPool
//The resource pool to which this virtual machine should be attached.
//    For a relocate or clone operation to a virtual machine, if the argument is not supplied, the current resource pool of virtual machine is used.
//    For a clone operation from a template to a virtual machine, this argument is required.
//    If the virtual machine is relocated to a different vCenter service, both the destination host and resource pool have to be specified and cannot be unset.
//    If the virtual machine is relocated to a different datacenter within the vCenter service, the resource pool has to be specified and cannot be unset.
func parseLocationPool(api *helper.API, p CloneParameter, location *types.VirtualMachineRelocateSpec) error {
	l := p.Location
	if l.ResourcePoolID != nil {
		oPool := resourcepool.GetObject(api, *l.ResourcePoolID)
		if oPool == nil {
			return fmt.Errorf("资源池[%s]无法找到", *l.ResourcePoolID)
		}
		poolRef := oPool.Reference()
		location.Pool = &poolRef
		return nil
	}
	if l.ClusterID != nil {
		moCluster := clustercomputerresource.GetMObject(api, *l.ClusterID)
		if moCluster == nil {
			return fmt.Errorf("集群[%s]无法找到", *l.ClusterID)
		}
		location.Pool = moCluster.ResourcePool
		return nil
	}
	if l.HostId != nil {
		location.Pool = hostsystem.GetParentPool(api, *l.HostId)
		return nil
	}
	if location.Pool == nil {
		return fmt.Errorf("缺少localtion参数，至少需要指定[资源池]、[集群]、[主机]其中一个")
	}
	return nil
}
