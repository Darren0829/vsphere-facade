package virtualmachinereconfig

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"time"
	"vsphere_api/app/logging"
	"vsphere_api/app/utils"
	"vsphere_api/helper"
	"vsphere_api/helper/disk"
	"vsphere_api/helper/network"
	"vsphere_api/helper/virtualmachine"
)

type ReconfigureParameter struct {
	Name       *string    `json:"name,omitempty"`
	CreateDate *time.Time `json:"-"`
	Annotation *string    `json:"annotation,omitempty"`

	Cpu    *CpuParameter    `json:"cpu,omitempty"`
	Memory *MemoryParameter `json:"memory,omitempty"`
	Disk   *DiskParameter   `json:"disk,omitempty"`
	Nic    *NicParameter    `json:"nic,omitempty"`
	Flag   *FlagParameter   `json:"flag,omitempty"`

	reboot bool
}

type FlagParameter struct {
	EnableLogging *bool `json:"enableLogging,omitempty"`
}

type CpuParameter struct {
	NumCPU            *int32 `json:"numCPU,omitempty"`
	NumCoresPerSocket *int32 `json:"numCoresPerSocket,omitempty"`

	CpuHotAddEnabled    *bool `json:"cpuHotAddEnabled,omitempty"`
	CpuHotRemoveEnabled *bool `json:"cpuHotRemoveEnabled,omitempty"`

	Allocation *AllocationParameter `json:"allocation,omitempty"`
}

type MemoryParameter struct {
	MemoryMB            *int32               `json:"memoryMB,omitempty"`
	MemoryHotAddEnabled *bool                `json:"memoryHotAddEnabled,omitempty"`
	Allocation          *AllocationParameter `json:"allocation,omitempty"`
}

type AllocationParameter struct {
	ExpandableReservation *bool   `json:"expandableReservation,omitempty"`
	Limit                 *int64  `json:"limit,omitempty"`
	OverheadLimit         *int64  `json:"overheadLimit,omitempty"`
	Reservation           *int64  `json:"reservation,omitempty"`
	Shares                *int32  `json:"shares,omitempty"`
	Level                 *string `json:"level,omitempty"`
}

type DiskParameter struct {
	Add    []*AddDiskParameter
	Edit   []EditDiskParameter
	Remove []int32
}

type AddDiskParameter struct {
	Size            int32
	Format          string
	Mode            *string
	StoragePolicyID *string
	Sharing         *string

	UnitNumber *int32 `json:"-"`
}

type RemoveDiskParameter struct {
	Key int32
}

type EditDiskParameter struct {
	Key             int32
	Size            *int32
	Mode            *string
	StoragePolicyID *string
	Sharing         *string
}

type AttachDiskParameter struct {
	Key int32
}

type DetachDiskParameter struct {
	Key int32
}

type NicParameter struct {
	Add    []*AddNicParameter
	Edit   []EditNicParameter
	Remove []int32
}

type AddNicParameter struct {
	ID string `json:"-"`

	NetworkID   string  `json:"networkId"`
	AdapterType *string `json:"adapterType"`
	AddressType *string `json:"-"`
	MACAddress  *string `json:"macAddress"`
	Allocation  *NetworkAllocation

	UnitNumber *int32 `json:"-"`
}

type NetworkAllocation struct {
	Limit       *int64  `json:"limit,omitempty"`
	Reservation *int64  `json:"reservation,omitempty"`
	Shares      *int32  `json:"shares,omitempty"`
	Level       *string `json:"level,omitempty"`
}

type RemoveNicParameter struct {
	Key int32
}

type EditNicParameter struct {
	Key        int32
	NetworkID  *string `json:"network_id"`
	MACAddress *string `json:"mac_address"`
}

func Reconfigure(api *helper.API, ID string, p *ReconfigureParameter) (*object.VirtualMachine, error) {
	logging.L().Debug(fmt.Sprintf("修改虚拟机[%s]配置", ID))
	oVM := virtualmachine.GetObject(api, ID)
	if oVM == nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败，虚拟机不存在", ID))
	}
	var spec types.VirtualMachineConfigSpec
	var err error
	err = parseReconfigureInfo(api, *p, &spec)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	err = parseReconfigureCpu(*p, &spec)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	err = parseReconfigureMemory(*p, &spec)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	err = parseFlag(*p, &spec)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	devices := virtualmachine.GetDevices(oVM)
	if devices == nil {
		return nil, fmt.Errorf(fmt.Sprintf("未获取到虚拟机[%s]的设备列表", ID))
	}
	var deviceChange []types.BaseVirtualDeviceConfigSpec
	err = parseReconfigureDisk(api, p, devices, &deviceChange)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	err = parseReconfigureNic(api, p, devices, &deviceChange)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}

	if len(deviceChange) > 0 {
		spec.DeviceChange = deviceChange
	}
	err = waitReconfigureTask(oVM, spec)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("修改虚拟机[%s]配置失败: %s", ID, err))
	}
	logging.L().Debug(fmt.Sprintf("虚拟机[%s]配置修改完成", ID))
	return oVM, nil
}

func waitReconfigureTask(oVM *object.VirtualMachine, spec types.VirtualMachineConfigSpec) error {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	task, err := oVM.Reconfigure(ctx, spec)
	if err != nil {
		return err
	}
	tctx, tcancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer tcancel()
	return task.Wait(tctx)
}

func parseReconfigureInfo(api *helper.API, p ReconfigureParameter, spec *types.VirtualMachineConfigSpec) error {
	if api.Newer(6, 7, 0) {
		if p.CreateDate != nil {
			spec.CreateDate = p.CreateDate
		}
	} else {
		logging.L().Debug("跳过设置[创建时间]")
	}
	if p.Name != nil {
		spec.Name = *p.Name
	}
	if p.Annotation != nil {
		spec.Annotation = *p.Annotation
	}
	return nil
}

func parseReconfigureCpu(p ReconfigureParameter, spec *types.VirtualMachineConfigSpec) error {
	cpu := p.Cpu
	if cpu == nil {
		return nil
	}
	if cpu.NumCPU != nil {
		spec.NumCPUs = *cpu.NumCPU
	}
	if cpu.NumCoresPerSocket != nil {
		spec.NumCoresPerSocket = *cpu.NumCoresPerSocket
	}
	if cpu.CpuHotAddEnabled != nil {
		spec.CpuHotAddEnabled = cpu.CpuHotAddEnabled
	}
	if cpu.CpuHotRemoveEnabled != nil {
		spec.CpuHotRemoveEnabled = cpu.CpuHotRemoveEnabled
	}

	a := cpu.Allocation
	if a != nil {
		var rai types.ResourceAllocationInfo
		{
		}
		if a.ExpandableReservation != nil {
			rai.ExpandableReservation = a.ExpandableReservation
		}
		if a.OverheadLimit != nil {
			rai.OverheadLimit = a.OverheadLimit
		}
		if a.Limit != nil {
			rai.Limit = a.Limit
		}
		if a.Reservation != nil {
			rai.Reservation = a.Reservation
		}
		if a.Level != nil {
			var si = types.SharesInfo{}
			si.Level = types.SharesLevel(*a.Level)
			if types.SharesLevelCustom == si.Level && a.Shares != nil {
				si.Shares = *a.Shares
			}
			rai.Shares = &si
		}
		spec.CpuAllocation = &rai
	}
	return nil
}

func parseReconfigureMemory(p ReconfigureParameter, spec *types.VirtualMachineConfigSpec) error {
	mem := p.Memory
	if mem == nil {
		return nil
	}
	if mem.MemoryMB != nil {
		spec.MemoryMB = int64(*mem.MemoryMB)
	}
	if mem.MemoryHotAddEnabled != nil {
		spec.MemoryHotAddEnabled = mem.MemoryHotAddEnabled
	}
	a := mem.Allocation
	if a != nil {
		var rai types.ResourceAllocationInfo
		{
		}
		if a.ExpandableReservation != nil {
			rai.ExpandableReservation = a.ExpandableReservation
		}
		if a.OverheadLimit != nil {
			rai.OverheadLimit = a.OverheadLimit
		}
		if a.Limit != nil {
			rai.Limit = a.Limit
		}
		if a.Reservation != nil {
			rai.Reservation = a.Reservation
		}
		if a.Level != nil {
			var si = types.SharesInfo{}
			si.Level = types.SharesLevel(*a.Level)
			if types.SharesLevelCustom == si.Level && a.Shares != nil {
				si.Shares = *a.Shares
			}
			rai.Shares = &si
		}
		spec.CpuAllocation = &rai
	}
	return nil
}

func parseReconfigureDisk(api *helper.API, p *ReconfigureParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if p.Disk == nil {
		return nil
	}

	// 移除
	err := removeDisk(p.Disk.Remove, devices, change)
	if err != nil {
		return fmt.Errorf("移除硬盘失败: %s", err)
	}
	// 修改
	err = editDisk(api, p.Disk.Edit, devices, change)
	if err != nil {
		return fmt.Errorf("修改硬盘失败: %s", err)
	}
	// 添加
	err = addDisk(api, p.Disk.Add, devices, change)
	if err != nil {
		return fmt.Errorf("添加硬盘失败: %s", err)
	}
	return nil
}

func parseFlag(p ReconfigureParameter, spec *types.VirtualMachineConfigSpec) error {
	if p.Flag == nil {
		return nil
	}

	changed := false
	flagInfo := types.VirtualMachineFlagInfo{}
	if p.Flag.EnableLogging != nil {
		flagInfo.EnableLogging = p.Flag.EnableLogging
		changed = true
	}

	if changed {
		spec.Flags = &flagInfo
	}
	return nil
}

func removeDisk(removeDisks []int32, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if len(removeDisks) == 0 {
		return nil
	}
	var removeKeys []int32
	for _, key := range removeDisks {
		removeKeys = append(removeKeys, key)
	}

	disks := devices.SelectByType((*types.VirtualDisk)(nil))
	for _, d := range disks {
		if *d.GetVirtualDevice().UnitNumber == int32(0) {
			// 系统盘不可移除，跳过
			continue
		}
		if utils.SliceContain(removeKeys, d.GetVirtualDevice().Key) {
			configSpec := &types.VirtualDeviceConfigSpec{
				Device:        d,
				FileOperation: types.VirtualDeviceConfigSpecFileOperationDestroy,
				Operation:     types.VirtualDeviceConfigSpecOperationRemove,
			}
			*change = append(*change, configSpec)
		}
	}
	return nil
}

func addDisk(api *helper.API, newDisks []*AddDiskParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if len(newDisks) == 0 {
		return nil
	}
	ctrl, err := devices.FindDiskController("")
	if err != nil {
		return fmt.Errorf("获取控制器失败: %v", err)
	}

	newUnitNums := generateUnitNum(devices, "disk", len(newDisks))
	for i, nd := range newDisks {
		vd := types.VirtualDisk{}
		size := int64(nd.Size) * int64(1024) * int64(1024)
		vd.CapacityInKB = size
		vd.CapacityInBytes = size * int64(1024)
		backing := types.VirtualDiskFlatVer2BackingInfo{}
		formatMapping := disk.FormatMapping[nd.Format]
		backing.EagerlyScrub = formatMapping.EagerlyScrub
		backing.ThinProvisioned = formatMapping.ThinProvisioned
		if nd.Mode != nil {
			backing.DiskMode = *nd.Mode
		}

		if api.Newer(6, 0, 0) {
			if nd.Sharing != nil {
				backing.Sharing = *nd.Sharing
			}
		} else {
			logging.L().Debug("跳过[Sharing]设置")
		}
		vd.Backing = &backing
		vd.UnitNumber = &newUnitNums[i]
		nd.UnitNumber = vd.UnitNumber
		configSpec := &types.VirtualDeviceConfigSpec{
			Device:        &vd,
			Operation:     types.VirtualDeviceConfigSpecOperationAdd,
			FileOperation: types.VirtualDeviceConfigSpecFileOperationCreate,
		}
		if nd.StoragePolicyID != nil {
			configSpec.Profile = []types.BaseVirtualMachineProfileSpec{
				&types.VirtualMachineDefinedProfileSpec{
					ProfileId: *nd.StoragePolicyID,
				},
			}
		}
		devices.AssignController(&vd, ctrl)
		*change = append(*change, configSpec)
	}
	return nil
}

func editDisk(api *helper.API, editDisks []EditDiskParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if len(editDisks) == 0 {
		return nil
	}
	disks := devices.SelectByType((*types.VirtualDisk)(nil))
	diskMap := make(map[int32]*types.VirtualDisk)
	for _, d := range disks {
		key := d.GetVirtualDevice().Key
		diskMap[key] = d.(*types.VirtualDisk)
	}

	for _, ed := range editDisks {
		key := ed.Key
		vd := diskMap[key]
		if vd != nil {
			if ed.Size != nil {
				newSize := int64(*ed.Size) * int64(1024) * int64(1024)
				vd.CapacityInBytes = newSize * int64(1024)
				vd.CapacityInKB = newSize
			}
			backing := vd.Backing.(*types.VirtualDiskFlatVer2BackingInfo)
			if ed.Mode != nil {
				backing.DiskMode = *ed.Mode
			}

			if api.Newer(6, 0, 0) {
				if ed.Sharing != nil {
					backing.Sharing = *ed.Sharing
				}
			} else {
				logging.L().Debug("跳过[Sharing]设置")
			}

			configSpec := &types.VirtualDeviceConfigSpec{
				Device:    vd,
				Operation: types.VirtualDeviceConfigSpecOperationEdit,
			}
			if ed.StoragePolicyID != nil {
				configSpec.Profile = []types.BaseVirtualMachineProfileSpec{
					&types.VirtualMachineDefinedProfileSpec{
						ProfileId: *ed.StoragePolicyID,
					},
				}
			}
			*change = append(*change, configSpec)
		} else {
			logging.L().Error("硬盘[%d]不存在", key)
		}
	}

	return nil
}

func parseReconfigureNic(api *helper.API, p *ReconfigureParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if p.Nic == nil {
		return nil
	}
	ethernetCards := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	if ethernetCards == nil {
		return nil
	}

	var err error
	// 移除
	err = removeNic(p.Nic.Remove, devices, change)
	if err != nil {
		return fmt.Errorf("移除网卡失败: %s", err)
	}
	// 修改
	err = editNic(api, p.Nic.Edit, devices, change)
	if err != nil {
		return fmt.Errorf("修改网卡失败: %s", err)
	}
	// 添加
	err = addNic(api, p.Nic.Add, devices, change)
	if err != nil {
		return fmt.Errorf("添加网卡失败: %s", err)
	}
	return nil
}

func removeNic(nics []int32, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if len(nics) == 0 {
		return nil
	}

	ethernetCards := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	for _, card := range ethernetCards {
		if utils.SliceContain(nics, card.GetVirtualDevice().Key) {
			configSpec := &types.VirtualDeviceConfigSpec{
				Device:    card,
				Operation: types.VirtualDeviceConfigSpecOperationRemove,
			}
			*change = append(*change, configSpec)
		}
	}
	return nil
}

func editNic(api *helper.API, nics []EditNicParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	if nics == nil {
		return nil
	}

	ethernetCards := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	var cardMap = make(map[int32]types.BaseVirtualDevice)
	for _, card := range ethernetCards {
		key := card.GetVirtualDevice().Key
		cardMap[key] = card.(types.BaseVirtualDevice)
	}

	for _, nic := range nics {
		editCard := cardMap[nic.Key]
		if editCard == nil {
			return fmt.Errorf("网卡[%d]不存在", nic.Key)
		}

		var destinationNetwork *mo.Network
		if nic.NetworkID != nil {
			destinationNetwork = network.GetMObject(api, *nic.NetworkID)
			if destinationNetwork == nil {
				return fmt.Errorf("网卡[%d]想要修改的目标网络[%s]不存在", nic.Key, *nic.NetworkID)
			}
		}
		changed := false
		switch editCard.(type) {
		case *types.VirtualE1000:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualE1000).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualE1000).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualE1000).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualE1000).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualE1000).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		case *types.VirtualE1000e:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualE1000e).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualE1000e).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualE1000e).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualE1000e).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualE1000e).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		case *types.VirtualPCNet32:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualPCNet32).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualPCNet32).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualPCNet32).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualPCNet32).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualPCNet32).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		case *types.VirtualVmxnet:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualVmxnet).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualVmxnet).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualVmxnet).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualVmxnet).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualVmxnet).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		case *types.VirtualVmxnet2:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualVmxnet2).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualVmxnet2).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualVmxnet2).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualVmxnet2).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualVmxnet2).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		case *types.VirtualVmxnet3:
			if destinationNetwork != nil {
				ref := destinationNetwork.Reference()
				backing := editCard.(*types.VirtualVmxnet3).Backing.(*types.VirtualEthernetCardNetworkBackingInfo)
				backing.Network = &ref
				backing.DeviceName = destinationNetwork.Name
				editCard.(*types.VirtualVmxnet3).Backing = backing
				changed = true
			}

			if nic.MACAddress != nil {
				macAddress := editCard.(*types.VirtualVmxnet3).MacAddress
				if macAddress != *nic.MACAddress {
					editCard.(*types.VirtualVmxnet3).AddressType = string(types.VirtualEthernetCardMacTypeManual)
					editCard.(*types.VirtualVmxnet3).MacAddress = *nic.MACAddress
					changed = true
				}
			}
		}

		if changed {
			dspec, _ := object.VirtualDeviceList{editCard}.ConfigSpec(types.VirtualDeviceConfigSpecOperationEdit)
			*change = append(*change, dspec...)
		}
	}
	return nil
}

func addNic(api *helper.API, nics []*AddNicParameter, devices object.VirtualDeviceList, change *[]types.BaseVirtualDeviceConfigSpec) error {
	// todo 分布式端口组类型的网卡创建
	if len(nics) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	nmap := make(map[string]*object.Network)

	newUnitNums := generateUnitNum(devices, "nic", len(nics))
	for i, nic := range nics {
		networkID := nic.NetworkID
		oNetwork := nmap[networkID]
		if oNetwork == nil {
			oNetwork = network.GetObject(api, nic.NetworkID)
			if oNetwork == nil {
				return fmt.Errorf("添加网卡失败: 网络[%s]不存在", nic.NetworkID)
			}
			nmap[networkID] = oNetwork
		}
		backing, err := oNetwork.EthernetCardBackingInfo(ctx)
		if err != nil {
			return fmt.Errorf("获取网络Backing出错: %s", err)
		}
		if nic.AdapterType == nil {
			return fmt.Errorf("创建网卡失败: 未设置[AdapterType]参数")
		}
		device, err := devices.CreateEthernetCard(*nic.AdapterType, backing)
		if err != nil {
			return fmt.Errorf("创建网卡失败: %s", err)
		}
		err = devices.Connect(device)
		if err != nil {
			logging.L().Error("设备连接失败", err)
		}
		card := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
		card.UnitNumber = &newUnitNums[i]
		nic.UnitNumber = card.UnitNumber

		if nic.AddressType != nil {
			card.AddressType = *nic.AddressType
			if nic.MACAddress != nil && card.AddressType == string(types.VirtualEthernetCardMacTypeManual) {
				card.MacAddress = *nic.MACAddress
			}
		}

		if api.Newer(6, 0, 0) {
			if nic.Allocation != nil {
				alloc := nic.Allocation
				card.ResourceAllocation = &types.VirtualEthernetCardResourceAllocation{
					Limit:       alloc.Limit,
					Reservation: alloc.Reservation,
					Share: types.SharesInfo{
						Shares: *alloc.Shares,
						Level:  types.SharesLevel(*alloc.Level),
					},
				}
			}
		} else {
			logging.L().Debug("VC版本低于6.0.0，跳过设置网卡[Allocation]")
		}

		dspec, _ := object.VirtualDeviceList{device}.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
		*change = append(*change, dspec...)
	}
	return nil
}

func generateUnitNum(devices object.VirtualDeviceList, deviceType string, cnt int) []int32 {
	var usedNu []int32
	for _, d := range devices {
		nu := d.GetVirtualDevice().UnitNumber
		if nu != nil {
			usedNu = append(usedNu, *nu)
		}
	}

	var startNn int32
	switch deviceType {
	case "disk":
		startNn = 1
		// 保留7
		usedNu = append(usedNu, 7)
	case "nic":
		startNn = 7
	}

	var unitNumbers []int32
	newNU := startNn
	for len(unitNumbers) < cnt {
		if utils.SliceContain(usedNu, newNU) {
			newNU++
			continue
		}
		unitNumbers = append(unitNumbers, newNU)
		newNU++
	}
	return unitNumbers
}
