package virtualmachinerelocate

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"strings"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
	"vsphere_api/helper/clustercomputerresource"
	"vsphere_api/helper/datastore"
	"vsphere_api/helper/disk"
	"vsphere_api/helper/hostsystem"
	"vsphere_api/helper/network"
	"vsphere_api/helper/resourcepool"
	"vsphere_api/helper/virtualmachine"
	"vsphere_api/helper/vsphere"
)

type RelocateParameter struct {
	Compute *ComputeParameter `json:"compute"`
	Storage *StorageParameter `json:"storage"`
}

type ComputeParameter struct {
	DestinationID *string           `json:"destinationId"`
	Network       *NetworkParameter `json:"network"`
}

type NetworkParameter struct {
	NetworkID *string               `json:"networkId"`
	Nics      []NicNetworkParameter `json:"nics"`
}

type NicNetworkParameter struct {
	Key       int32  `json:"key"`
	NetworkID string `json:"network_id"`
}

type StorageParameter struct {
	StoragePolicyID *string                `json:"storagePolicyId"`
	DatastoreID     *string                `json:"datastoreId"`
	Disks           []DiskStorageParameter `json:"disks"`
}

type DiskStorageParameter struct {
	Key             int32   `json:"key"`
	DatastoreID     *string `json:"datastoreId"`
	Format          *string `json:"format"`
	StoragePolicyID *string `json:"storagePolicyId"`
}

func Relocate(api *helper.API, ID string, p RelocateParameter, timeout int32) (*object.VirtualMachine, error) {
	logging.L().Debug("迁移开始")
	oVM := virtualmachine.GetObject(api, ID)
	if oVM == nil {
		return nil, fmt.Errorf("虚拟机[%s]无法找到", ID)
	}

	var err error
	var hasChange bool
	var config types.VirtualMachineRelocateSpec
	computeChanged, err := parseCompute(api, oVM, p, &config)
	if err != nil {
		return nil, fmt.Errorf("虚拟机[%s(%s)]迁移失败: %s", ID, oVM.Name(), err)
	}
	if !hasChange {
		hasChange = computeChanged
	}

	storageChanged, err := parseStorage(api, oVM, p, &config)
	if err != nil {
		return nil, fmt.Errorf("虚拟机[%s(%s)]迁移失败: %s", ID, oVM.Name(), err)
	}
	if !hasChange {
		hasChange = storageChanged
	}

	if hasChange {
		ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
		defer cancel()
		oTask, err := oVM.Relocate(ctx, config, types.VirtualMachineMovePriorityDefaultPriority)
		if err != nil {
			return nil, fmt.Errorf("虚拟机[%s(%s)]迁移失败: %s", ID, oVM.Name(), err)
		}
		err = vsphere.WaitForTask(oTask, timeout)
		if err != nil {
			return nil, fmt.Errorf("虚拟机[%s(%s)]迁移失败: %s", ID, oVM.Name(), err)
		}
	}
	oVM = virtualmachine.GetObject(api, ID)
	return oVM, nil
}

func parseCompute(api *helper.API, oVM *object.VirtualMachine, p RelocateParameter, spec *types.VirtualMachineRelocateSpec) (bool, error) {
	if p.Compute == nil || p.Compute.DestinationID == nil {
		return false, nil
	}
	hasChange := false
	location := virtualmachine.GetLocationInfo(api, oVM)
	destinationID := *p.Compute.DestinationID
	rtype := strings.Split(destinationID, "-")[0]
	switch rtype {
	case "domain":
		if location.ClusterID != destinationID {
			cluster := clustercomputerresource.GetMObject(api, destinationID)
			if cluster == nil {
				return false, fmt.Errorf("要迁移至的目标集群p[%s]不存在", destinationID)
			}
			// 集群也是资源池
			spec.Pool = &types.ManagedObjectReference{
				Type:  resourcepool.Type,
				Value: cluster.ResourcePool.Reference().Value,
			}
			hasChange = true
		}
	case "host":
		if location.HostID != destinationID {
			host := hostsystem.GetObject(api, destinationID)
			if host == nil {
				return false, fmt.Errorf("要迁移至的目标主机[%s]不存在", destinationID)
			}
			spec.Host = &types.ManagedObjectReference{
				Type:  hostsystem.Type,
				Value: destinationID,
			}
			hasChange = true
		}
	case "resgroup":
		if location.ResourcePoolID != destinationID {
			pool := resourcepool.GetObject(api, destinationID)
			if pool == nil {
				return false, fmt.Errorf("要迁移至的目标资源池/vApp[%s]不存在", destinationID)
			}
			spec.Pool = &types.ManagedObjectReference{
				Type:  resourcepool.Type,
				Value: destinationID,
			}
			hasChange = true
		}
	default:
		return false, fmt.Errorf("无法识别的计算资源ID: %s", destinationID)
	}

	if hasChange {
		_, err := parseNetwork(api, oVM, p.Compute.Network, spec)
		if err != nil {
			return false, fmt.Errorf("虚拟机[%s(%s)]迁移失败: %s", oVM.Reference().Value, oVM.Name(), err)
		}
	}
	return hasChange, nil
}

func parseStorage(api *helper.API, oVM *object.VirtualMachine, p RelocateParameter, spec *types.VirtualMachineRelocateSpec) (bool, error) {
	if p.Storage == nil {
		return false, nil
	}
	hasChange := false
	if p.Storage.DatastoreID != nil {
		oDatastore := datastore.GetObject(api, *p.Storage.DatastoreID)
		if oDatastore != nil {
			datastoreRef := oDatastore.Reference()
			spec.Datastore = &datastoreRef
			hasChange = true
		} else {
			return false, fmt.Errorf("存储[%s]不存在", *p.Storage.DatastoreID)
		}
	}

	if p.Storage.StoragePolicyID != nil {
		spec.Profile = []types.BaseVirtualMachineProfileSpec{
			&types.VirtualMachineDefinedProfileSpec{
				ProfileId: *p.Storage.StoragePolicyID,
			},
		}
		hasChange = true
	}

	if p.Storage.Disks != nil {
		var diskMap = make(map[int32]*types.VirtualDisk)
		disks := virtualmachine.GetDisks(oVM)
		for _, diskDevice := range disks {
			key := diskDevice.GetVirtualDevice().Key
			diskMap[key] = diskDevice.(*types.VirtualDisk)
		}

		var diskLocators []types.VirtualMachineRelocateSpecDiskLocator
		for _, d := range p.Storage.Disks {
			dp := diskMap[d.Key]
			if dp == nil {
				return false, fmt.Errorf("虚拟机[%s(%s)]硬盘[%d]不存在", oVM.Name(), oVM.Reference().Value, d.Key)
			}

			if d.DatastoreID == nil {
				continue
			}

			oDatastore := datastore.GetObject(api, *d.DatastoreID)
			if oDatastore == nil {
				return false, fmt.Errorf("存储[%s]不存在", *p.Storage.DatastoreID)
			}

			var diskLocator types.VirtualMachineRelocateSpecDiskLocator
			diskLocator.DiskId = d.Key
			diskLocator.Datastore = oDatastore.Reference()

			backing := dp.Backing
			switch backing.(type) {
			case *types.VirtualDiskFlatVer2BackingInfo:
				if d.Format != nil {
					formatMapping := disk.FormatMapping[*d.Format]
					backing.(*types.VirtualDiskFlatVer2BackingInfo).ThinProvisioned = formatMapping.ThinProvisioned
					backing.(*types.VirtualDiskFlatVer2BackingInfo).EagerlyScrub = formatMapping.EagerlyScrub
				}
			}
			diskLocator.DiskBackingInfo = backing

			if d.StoragePolicyID != nil {
				diskLocator.Profile = []types.BaseVirtualMachineProfileSpec{
					&types.VirtualMachineDefinedProfileSpec{
						ProfileId: *d.StoragePolicyID,
					},
				}
			}
			diskLocators = append(diskLocators, diskLocator)
		}

		if len(diskLocators) > 0 {
			spec.Disk = diskLocators
			hasChange = true
		}
	}
	return hasChange, nil
}

func parseNetwork(api *helper.API, oVM *object.VirtualMachine, p *NetworkParameter, spec *types.VirtualMachineRelocateSpec) (bool, error) {
	if p == nil {
		return false, nil
	}
	ethernetCards := virtualmachine.GetEthernetCards(oVM)
	if ethernetCards == nil {
		return false, fmt.Errorf("虚拟机[%s(%s)]没有网卡无法进行网络迁移", oVM.Name(), oVM.Reference().Value)
	}

	var destinationNetwork *mo.Network
	if p.NetworkID != nil {
		destinationNetwork = network.GetMObject(api, *p.NetworkID)
		if destinationNetwork == nil {
			return false, fmt.Errorf("想要迁移至的目标网络[%s]不存在", *p.NetworkID)
		}
	}

	parameterMap := make(map[int32]NicNetworkParameter)
	if p.Nics != nil {
		for _, nic := range p.Nics {
			key := nic.Key
			parameterMap[key] = nic
		}
	}

	var deviceChange []types.BaseVirtualDeviceConfigSpec
	for _, card := range ethernetCards {
		key := card.GetVirtualDevice().Key
		parameter := parameterMap[key]
		if parameter.NetworkID != "" {
			oNetwork := network.GetObject(api, parameter.NetworkID)
			if oNetwork == nil {
				return false, fmt.Errorf("想要迁移至的目标网络[%s]不存在", parameter.NetworkID)
			}
			ref := oNetwork.Reference()
			backing := card.(*types.VirtualEthernetCard).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
			backing.Network = &ref
			backing.DeviceName = oNetwork.Name()

			change, _ := object.VirtualDeviceList{card}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
			deviceChange = append(deviceChange, change...)
		} else if destinationNetwork != nil {
			ref := destinationNetwork.Reference()
			backing := card.(*types.VirtualEthernetCard).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
			backing.Network = &ref
			backing.DeviceName = destinationNetwork.Name

			change, _ := object.VirtualDeviceList{card}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
			deviceChange = append(deviceChange, change...)
		}
	}

	if deviceChange != nil {
		spec.DeviceChange = deviceChange
		return true, nil
	}
	return false, nil
}
