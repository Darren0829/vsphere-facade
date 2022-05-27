package workerpool

import (
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"vsphere_api/app/logging"
	"vsphere_api/config"
	"vsphere_api/helper"
	"vsphere_api/helper/virtualmachine"
	"vsphere_api/helper/virtualmachine/virtualmachinecustomize"
	"vsphere_api/helper/virtualmachine/virtualmachinereconfig"
	"vsphere_api/helper/virtualmachine/virtualmachinerelocate"
)

type VirtualMachineOperator struct {
	api     *helper.API
	oVM     *object.VirtualMachine
	id      string
	name    string
	display string
}

func GetVirtualMachineOperator(api *helper.API, ID string) *VirtualMachineOperator {
	oVM := virtualmachine.GetObject(api, ID)
	if oVM == nil {
		return nil
	}

	return &VirtualMachineOperator{
		api:     api,
		oVM:     oVM,
		id:      ID,
		name:    oVM.Name(),
		display: fmt.Sprintf("%s(%s)", oVM.Name(), oVM.Reference().Value),
	}
}

func (o VirtualMachineOperator) PowerOn() error {
	err := virtualmachine.PowerOn(o.oVM, true, 2)
	return err
}

func (o VirtualMachineOperator) PowerOff() error {
	if config.G.Vsphere.Default.Operation.ShutdownFirst {
		err := virtualmachine.Shutdown(o.oVM, true, 2)
		if err == nil {
			return nil
		}
	}
	err := virtualmachine.PowerOff(o.oVM, true, 2)
	return err
}

func (o VirtualMachineOperator) Shutdown() error {
	err := virtualmachine.Shutdown(o.oVM, true, 2)
	return err
}

func (o VirtualMachineOperator) Restart() error {
	err := virtualmachine.Shutdown(o.oVM, true, 2)
	if err != nil {
		logging.L().Warn(fmt.Sprintf("关闭虚拟机[%s]操作系统失败，尝试关闭电源", o.display), err)
		err = virtualmachine.PowerOff(o.oVM, true, 2)
	}
	err = virtualmachine.PowerOn(o.oVM, true, 2)
	if err != nil {
		logging.L().Error(fmt.Sprintf("重启虚拟机[%s]失败", o.display), err)
	}
	return err
}

func (o VirtualMachineOperator) Destroy() error {
	err := virtualmachine.Destroy(o.api, o.id)
	if err != nil {
		logging.L().Error(fmt.Sprintf("删除虚拟机[%s]失败", o.display))
	}
	return err
}

func (o VirtualMachineOperator) CreateSnapshot() error {
	// todo 创建快照
	return nil
}

func (o VirtualMachineOperator) Rename(newName string) error {
	if o.oVM.Name() == newName {
		return nil
	}
	return virtualmachine.Rename(o.oVM, newName, 10)
}

func (o VirtualMachineOperator) Descript(annotation string) error {
	p := &virtualmachinereconfig.ReconfigureParameter{Annotation: &annotation}
	_, err := virtualmachinereconfig.Reconfigure(o.api, o.id, p)
	return err
}

type RelocateParameter struct {
	Compute *virtualmachinerelocate.ComputeParameter `json:"compute"`
	Storage *virtualmachinerelocate.StorageParameter `json:"storage"`
}

func (o VirtualMachineOperator) Relocate(p RelocateParameter) error {
	parameter := virtualmachinerelocate.RelocateParameter{
		Compute: p.Compute,
		Storage: p.Storage,
	}
	_, err := virtualmachinerelocate.Relocate(o.api, o.id, parameter, config.G.Vsphere.Timeout.WaitForRelocate)
	return err
}

type ReconfigureParameter struct {
	NumCPU            int32 `json:"numCPU"`
	NumCoresPerSocket int32 `json:"numCoresPerSocket"`
	MemoryMB          int32 `json:"MemoryMB"`
}

func (o VirtualMachineOperator) Reconfigure(p ReconfigureParameter) error {
	props := []string{"config.hardware.memoryMB", "config.hardware.numCPU", "config.hardware.numCoresPerSocket",
		"runtime.powerState", "config.memoryHotAddEnabled", "config.cpuHotAddEnabled", "config.cpuHotRemoveEnabled"}
	moVM := virtualmachine.FindProps(o.oVM, props...)

	var cpuChanged, memoryChanged, addCPU, removeCPU, addMemory, removeMemory bool
	var parameter virtualmachinereconfig.ReconfigureParameter
	if p.NumCPU > 0 || p.NumCoresPerSocket > 0 {
		currentNumCPU := moVM.Config.Hardware.NumCPU
		cpuParameter := virtualmachinereconfig.CpuParameter{}
		if p.NumCPU != currentNumCPU {
			cpuParameter.NumCPU = &p.NumCPU
			addCPU = p.NumCPU > currentNumCPU
			removeCPU = p.NumCPU < currentNumCPU
		}

		if p.NumCoresPerSocket != moVM.Config.Hardware.NumCoresPerSocket {
			cpuParameter.NumCoresPerSocket = &p.NumCoresPerSocket
			cpuChanged = true
		}

		if cpuChanged {
			parameter.Cpu = &cpuParameter
		}
	}

	currentMemoryMB := moVM.Config.Hardware.MemoryMB
	if p.MemoryMB > 0 && p.MemoryMB != currentMemoryMB {
		parameter.Memory = &virtualmachinereconfig.MemoryParameter{
			MemoryMB: &p.MemoryMB,
		}
		addMemory = p.MemoryMB > currentMemoryMB
		removeMemory = p.MemoryMB < currentMemoryMB
		memoryChanged = true
	}

	if !cpuChanged && !memoryChanged {
		logging.L().Warnf("虚拟机[%s]配置未发生改变，配置修改中断", o.display)
		return nil
	}

	isPoweredOn := moVM.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn
	// 不支持热插拔就需要先关机
	stopFirst := false
	if isPoweredOn {
		memoryHotAddEnabled := moVM.Config.MemoryHotAddEnabled
		cpuHotAddEnabled := moVM.Config.CpuHotAddEnabled
		cpuHotRemoveEnabled := moVM.Config.CpuHotRemoveEnabled
		if addMemory && memoryHotAddEnabled != nil && !*memoryHotAddEnabled {
			stopFirst = true
		} else if removeMemory {
			stopFirst = true
		} else if addCPU && cpuHotAddEnabled != nil && !*cpuHotAddEnabled {
			stopFirst = true
		} else if removeCPU && cpuHotRemoveEnabled != nil && !*cpuHotRemoveEnabled {
			stopFirst = true
		}
	}

	if stopFirst {
		err := o.PowerOff()
		if err != nil {
			return fmt.Errorf("虚拟机[%s]关机失败: %v", o.display, err)
		}
	}

	_, err := virtualmachinereconfig.Reconfigure(o.api, o.id, &parameter)
	if err != nil {
		return err
	}

	if stopFirst && !isPoweredOn {
		err := o.PowerOn()
		if err != nil {
			logging.L().Errorf("虚拟机[%s]启动失败", o.display)
		}
	}
	return err
}

func (o VirtualMachineOperator) ReconfigureCheck(p virtualmachinereconfig.DiskParameter) error {
	// todo 配置修改校验
	return nil
}

type ReconfigureDiskParameter struct {
	Add []struct {
		DatastoreID string `json:"datastoreId"`
		virtualmachinereconfig.AddDiskParameter
	}
	Edit   []virtualmachinereconfig.EditDiskParameter
	Remove []int32
}

func (o VirtualMachineOperator) ReconfigureDisk(p ReconfigureDiskParameter) error {
	reconfigureParameter := virtualmachinereconfig.ReconfigureParameter{}
	reconfigureParameter.Disk = &virtualmachinereconfig.DiskParameter{
		Edit:   p.Edit,
		Remove: p.Remove,
	}

	var mayRelocate []int
	var addDisks []*virtualmachinereconfig.AddDiskParameter
	for i, a := range p.Add {
		addDiskParameter := virtualmachinereconfig.AddDiskParameter{
			Size:            a.Size,
			Format:          a.Format,
			Mode:            a.Mode,
			StoragePolicyID: a.StoragePolicyID,
			Sharing:         a.Sharing,
		}
		addDisks = append(addDisks, &addDiskParameter)

		// 记录设置了datastore的硬盘
		if a.DatastoreID != "" {
			mayRelocate = append(mayRelocate, i)
		}
	}
	if addDisks != nil {
		reconfigureParameter.Disk.Add = addDisks
	}

	oVM, err := virtualmachinereconfig.Reconfigure(o.api, o.id, &reconfigureParameter)
	if err != nil {
		return err
	}
	o.oVM = oVM

	if len(mayRelocate) > 0 {
		disks := virtualmachine.GetDisks(o.oVM)

		diskUmMap := make(map[int32]*types.VirtualDisk)
		for _, d := range disks {
			un := *d.GetVirtualDevice().UnitNumber
			diskUmMap[un] = d.(*types.VirtualDisk)
		}

		var relocateDisks []virtualmachinerelocate.DiskStorageParameter
		for _, i := range mayRelocate {
			newDisk := addDisks[i]
			un := *newDisk.UnitNumber
			disk := diskUmMap[un]
			datastoreRef := disk.Backing.(*types.VirtualDiskFlatVer2BackingInfo).Datastore
			if p.Add[i].DatastoreID != datastoreRef.Value {
				diskStorageParameter := virtualmachinerelocate.DiskStorageParameter{
					Key:         disk.Key,
					DatastoreID: &p.Add[i].DatastoreID,
				}
				relocateDisks = append(relocateDisks, diskStorageParameter)
			}
		}

		if len(relocateDisks) > 0 {
			relocateParameter := virtualmachinerelocate.RelocateParameter{}
			relocateParameter.Storage = &virtualmachinerelocate.StorageParameter{
				Disks: relocateDisks,
			}
			oVM, err = virtualmachinerelocate.Relocate(o.api, o.id, relocateParameter, 10)
			if err != nil {
				return err
			}
			o.oVM = oVM
		}
	}
	return err
}

func (o VirtualMachineOperator) ReconfigureDiskCheck(p virtualmachinereconfig.DiskParameter) error {
	// todo 硬盘修改校验
	return nil
}

type ReconfigureNicParameter struct {
	Add []struct {
		DnsServerList []string                                `json:"dnsServerList"`
		DnsDomain     *string                                 `json:"dnsDomain"`
		Gateway       []string                                `json:"gateway"`
		SubnetMask    *int32                                  `json:"subnetMask"`
		IPv4          *virtualmachinecustomize.NicIPv4Setting `json:"ipv4"`
		IPv6          *virtualmachinecustomize.NicIPv6Setting `json:"ipv6"`
		virtualmachinereconfig.AddNicParameter
	}
	Edit []struct {
		DnsServerList []string                                `json:"dnsServerList"`
		DnsDomain     *string                                 `json:"dnsDomain"`
		Gateway       []string                                `json:"gateway"`
		SubnetMask    *int32                                  `json:"subnetMask"`
		IPv4          *virtualmachinecustomize.NicIPv4Setting `json:"ipv4"`
		IPv6          *virtualmachinecustomize.NicIPv6Setting `json:"ipv6"`
		virtualmachinereconfig.EditNicParameter
	}
	Remove []int32
}

func (o VirtualMachineOperator) ReconfigureNic(p ReconfigureNicParameter) error {
	var addNics []*virtualmachinereconfig.AddNicParameter
	for _, a := range p.Add {
		addNicParameter := virtualmachinereconfig.AddNicParameter{
			NetworkID:   a.NetworkID,
			AdapterType: a.AdapterType,
			Allocation:  a.Allocation,
		}
		if a.MACAddress != nil {
			addressType := string(types.VirtualEthernetCardMacTypeManual)
			addNicParameter.AddressType = &addressType
			addNicParameter.MACAddress = a.MACAddress
		}
		addNics = append(addNics, &addNicParameter)
	}

	var editNics []virtualmachinereconfig.EditNicParameter
	for _, e := range p.Edit {
		// 网卡
		editNicParameter := virtualmachinereconfig.EditNicParameter{
			Key:        e.Key,
			NetworkID:  e.NetworkID,
			MACAddress: e.MACAddress,
		}
		editNics = append(editNics, editNicParameter)
	}

	reconfigureParameter := virtualmachinereconfig.ReconfigureParameter{}
	reconfigureParameter.Nic = &virtualmachinereconfig.NicParameter{
		Add:    addNics,
		Edit:   editNics,
		Remove: p.Remove,
	}
	oVM, err := virtualmachinereconfig.Reconfigure(o.api, o.id, &reconfigureParameter)
	if err != nil {
		return err
	}
	o.oVM = oVM

	// 配置IP(不能配置IP，因为sdk必须同时修改所有网卡)
	//var nicSetting []*virtualmachinecustomize.NicSetting
	//nicDevices := virtualmachine.GetEthernetCards(o.oVM)
	//var keyMap = make(map[int32]int32)
	//for _, d := range nicDevices {
	//	un := *d.GetVirtualDevice().UnitNumber
	//	keyMap[un] = d.GetVirtualDevice().Key
	//}
	//
	//for i, a := range p.Add {
	//	// IP
	//	if a.IPv4 != nil || a.IPv6 != nil {
	//		un := *addNics[i].UnitNumber
	//		nicSetting = append(nicSetting, &virtualmachinecustomize.NicSetting{
	//			Key:           keyMap[un],
	//			DnsServerList: a.DnsServerList,
	//			DnsDomain:     a.DnsDomain,
	//			Gateway:       a.Gateway,
	//			SubnetMask:    a.SubnetMask,
	//			IPv4:          a.IPv4,
	//			IPv6:          a.IPv6,
	//		})
	//	}
	//}
	//
	//for _, e := range p.Edit {
	//	// IP
	//	if e.IPv4 != nil || e.IPv6 != nil {
	//		nicSetting = append(nicSetting, &virtualmachinecustomize.NicSetting{
	//			Key:           e.Key,
	//			DnsServerList: e.DnsServerList,
	//			DnsDomain:     e.DnsDomain,
	//			Gateway:       e.Gateway,
	//			SubnetMask:    e.SubnetMask,
	//			IPv4:          e.IPv4,
	//			IPv6:          e.IPv6,
	//		})
	//	}
	//}
	//
	//if len(nicSetting) > 0 {
	//	if virtualmachine.IsLinux(o.api, o.oVM) {
	//		hostname := virtualmachine.GetHostname(o.oVM)
	//		if hostname != "" {
	//			var customizeParameter virtualmachinecustomize.CustomizeParameter
	//			customizeParameter.OSSetting = &virtualmachinecustomize.OSSettingParameter{
	//				LinuxSetting: &virtualmachinecustomize.LinuxSettingParameter{
	//					HostName: &hostname,
	//				},
	//			}
	//			customizeParameter.NicSetting = nicSetting
	//			err = virtualmachinecustomize.Customize(o.api, o.id, &customizeParameter)
	//		} else {
	//			return fmt.Errorf("无法配置IP")
	//		}
	//	} else {
	//
	//	}
	//}
	return err
}

func (o VirtualMachineOperator) ReconfigureNicCheck(p virtualmachinereconfig.DiskParameter) error {
	// todo 网卡修改校验
	return nil
}
