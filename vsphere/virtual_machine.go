package vsphere

import (
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"strings"
	"vsphere_api/app/logging"
	"vsphere_api/helper/disk"
	"vsphere_api/helper/hostsystem"
	"vsphere_api/helper/virtualmachine"
	"vsphere_api/vsphere/protocol"
)

func (vc *VCenter) QueryVirtualMachines(q protocol.VirtualMachineQuery) []protocol.VirtualMachineInfo {
	if q.DatacenterID != "" {
		return vc.getVirtualMachinesByDatacenterID(q.DatacenterID)
	} else if q.FolderID != "" {
		return vc.getVirtualMachinesByFolderID(q.FolderID)
	} else if len(q.IDs) > 0 {
		return vc.getVirtualMachinesIDs(q.IDs)
	} else {
		return vc.getAllVirtualMachines()
	}
}

func (vc *VCenter) GetVirtualMachine(ID string) *protocol.VirtualMachineInfo {
	moVM := virtualmachine.GetMObject(vc.Api, ID)
	if moVM == nil {
		return nil
	}
	info := vc.buildVirtualMachineInfo(*moVM, "")
	return &info
}

func (vc *VCenter) getVirtualMachinesByDatacenterID(datacenterID string) []protocol.VirtualMachineInfo {
	var virtualMachineInfos []protocol.VirtualMachineInfo
	moVMs := virtualmachine.GetVirtualMachinesByDatacenterID(vc.Api, datacenterID)
	for _, vm := range moVMs {
		info := vc.buildVirtualMachineInfo(vm, "")
		virtualMachineInfos = append(virtualMachineInfos, info)
	}
	return virtualMachineInfos
}

func (vc *VCenter) getVirtualMachinesByFolderID(folderID string) []protocol.VirtualMachineInfo {
	var virtualMachineInfos []protocol.VirtualMachineInfo
	moVMs := virtualmachine.GetVirtualMachinesByFolderID(vc.Api, folderID)
	for _, vm := range moVMs {
		info := vc.buildVirtualMachineInfo(vm, "")
		virtualMachineInfos = append(virtualMachineInfos, info)
	}
	return virtualMachineInfos
}

func (vc *VCenter) getVirtualMachinesIDs(folderIDs []string) []protocol.VirtualMachineInfo {
	var virtualMachineInfos []protocol.VirtualMachineInfo
	folders := vc.getFoldersByIDs(folderIDs)
	for _, f := range folders {
		moVM := virtualmachine.GetMObject(vc.Api, f.ID)
		info := vc.buildVirtualMachineInfo(*moVM, f.DatacenterID)
		virtualMachineInfos = append(virtualMachineInfos, info)
	}
	return virtualMachineInfos
}

func (vc *VCenter) getAllVirtualMachines() []protocol.VirtualMachineInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}

	var virtualMachineInfos []protocol.VirtualMachineInfo
	for _, dc := range dcs {
		dcID := dc.ID
		vms := virtualmachine.GetVirtualMachinesByDatacenterID(vc.Api, dcID)
		if vms == nil {
			continue
		}
		for _, vm := range vms {
			info := vc.buildVirtualMachineInfo(vm, dcID)
			virtualMachineInfos = append(virtualMachineInfos, info)
		}
	}
	return virtualMachineInfos
}

func (vc *VCenter) buildVirtualMachineInfo(moVM mo.VirtualMachine, datacenterID string) protocol.VirtualMachineInfo {
	logging.L().Debug(fmt.Sprintf("构建虚拟机[%s(%s)]信息", moVM.Name, moVM.Reference().Value))
	info := protocol.VirtualMachineInfo{}
	info.ID = moVM.Reference().Value
	info.Name = moVM.Config.Name
	info.UUID = moVM.Config.Uuid
	info.InstanceUUID = moVM.Config.InstanceUuid
	info.Description = moVM.Config.Annotation
	info.CreateDate = moVM.Config.CreateDate

	info.NumCPU = moVM.Config.Hardware.NumCPU
	info.NumCoresPerSocket = moVM.Config.Hardware.NumCoresPerSocket
	info.MemoryMB = moVM.Config.Hardware.MemoryMB

	info.PowerState = string(moVM.Runtime.PowerState)
	info.IPAddress = moVM.Guest.IpAddress
	info.Hostname = moVM.Guest.HostName
	info.ToolsStatus = string(moVM.Guest.ToolsStatus)

	info.DatacenterID = datacenterID
	info.FolderID = vc.findFolder(moVM)
	info.HostID = moVM.Runtime.Host.Value
	info.ClusterID = vc.findCluster(moVM)
	info.ResourcePoolID = vc.findResourcePool(moVM)

	info.OSFamily = vc.getOSFamily(moVM.Config.GuestId)
	info.OSName = moVM.Config.GuestFullName
	if moVM.Config.Tools != nil {
		info.ToolsHasInstalled = moVM.Config.Tools.ToolsInstallType != ""
	}

	sysDisk, dataDisks := vc.findDisks(moVM)
	info.SysDisk = sysDisk
	info.DataDisks = dataDisks
	info.NetworkInterfaces = vc.findNetworkInterfaces(moVM)
	return info
}

func (vc *VCenter) QueryTemplates(q protocol.TemplateQuery) []protocol.TemplateInfo {
	if q.DatacenterID != "" {
		return vc.getTemplatesByDatacenterID(q.DatacenterID)
	} else if q.FolderID != "" {
		return vc.getTemplatesByFolderID(q.FolderID)
	} else if len(q.IDs) > 0 {
		return vc.getTemplatesIDs(q.IDs)
	} else {
		return vc.getAllTemplates()
	}
}

func (vc *VCenter) getTemplatesByDatacenterID(datacenterID string) []protocol.TemplateInfo {
	var templateInfos []protocol.TemplateInfo
	moVMs := virtualmachine.GetTemplatesByDatacenterID(vc.Api, datacenterID)
	for _, vm := range moVMs {
		templateInfo := vc.buildTemplateInfo(vm, "")
		templateInfos = append(templateInfos, templateInfo)
	}
	return templateInfos
}

func (vc *VCenter) getTemplatesByFolderID(folderID string) []protocol.TemplateInfo {
	var templateInfos []protocol.TemplateInfo
	moVMs := virtualmachine.GetTemplatesByFolderID(vc.Api, folderID)
	for _, vm := range moVMs {
		templateInfo := vc.buildTemplateInfo(vm, "")
		templateInfos = append(templateInfos, templateInfo)
	}
	return templateInfos
}

func (vc *VCenter) getTemplatesIDs(folderIDs []string) []protocol.TemplateInfo {
	var templateInfos []protocol.TemplateInfo
	folders := vc.getFoldersByIDs(folderIDs)
	for _, f := range folders {
		moVM := virtualmachine.GetMObject(vc.Api, f.ID)
		templateInfo := vc.buildTemplateInfo(*moVM, f.DatacenterID)
		templateInfos = append(templateInfos, templateInfo)
	}
	return templateInfos
}

func (vc *VCenter) getAllTemplates() []protocol.TemplateInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}

	var templateInfos []protocol.TemplateInfo
	for _, dc := range dcs {
		dcID := dc.ID
		vms := virtualmachine.GetTemplatesByDatacenterID(vc.Api, dcID)
		if vms == nil {
			continue
		}
		for _, vm := range vms {
			info := vc.buildTemplateInfo(vm, dcID)
			templateInfos = append(templateInfos, info)
		}
	}
	return templateInfos
}

func (vc *VCenter) buildTemplateInfo(moVM mo.VirtualMachine, datacenterID string) protocol.TemplateInfo {
	info := protocol.TemplateInfo{}
	info.ID = moVM.Reference().Value
	info.Name = moVM.Config.Name
	info.UUID = moVM.Config.Uuid
	info.InstanceUUID = moVM.Config.InstanceUuid
	info.Description = moVM.Config.Annotation
	info.CreateDate = moVM.Config.CreateDate

	info.NumCPU = moVM.Config.Hardware.NumCPU
	info.NumCoresPerSocket = moVM.Config.Hardware.NumCoresPerSocket
	info.MemoryMB = moVM.Config.Hardware.MemoryMB

	info.DatacenterID = datacenterID
	info.FolderID = vc.findFolder(moVM)
	info.HostID = moVM.Runtime.Host.Value
	info.ClusterID = vc.findCluster(moVM)
	info.ResourcePoolID = vc.findResourcePool(moVM)

	info.OSFamily = vc.getOSFamily(moVM.Config.GuestId)
	info.OSName = moVM.Config.GuestFullName
	if moVM.Config.Tools != nil {
		info.ToolsHasInstalled = moVM.Config.Tools.ToolsInstallType != ""
	}

	sysDisk, dataDisks := vc.findDisks(moVM)
	info.SysDisk = sysDisk
	info.DataDisks = dataDisks
	info.NetworkInterfaces = vc.findNetworkInterfaces(moVM)
	return info
}

func (vc *VCenter) getOSFamily(guestId string) string {
	if strings.Contains(guestId, "windows") {
		return "windows"
	} else {
		return "linux"
	}
}

func (vc *VCenter) findFolder(moVM mo.VirtualMachine) *string {
	return nil
}

func (vc *VCenter) findCluster(moVM mo.VirtualMachine) *string {
	hostID := moVM.Runtime.Host.Value
	host := vc.Cache.GetHost(hostID)
	if host != nil {
		return &host.ClusterID
	}

	cluster := hostsystem.GetCluster(vc.Api, hostID)
	if cluster != nil {
		clusterId := cluster.Reference().Value
		return &clusterId
	}
	return nil
}

func (vc *VCenter) findResourcePool(moVM mo.VirtualMachine) *string {
	return nil
}

func (vc *VCenter) findDatastore(moVM mo.VirtualMachine) string {
	return ""
}

func (vc *VCenter) findDisks(moVM mo.VirtualMachine) (protocol.DiskInfo, []protocol.DiskInfo) {
	var sysDisk protocol.DiskInfo
	var dataDisks []protocol.DiskInfo

	vmID := moVM.Reference().Value
	device := moVM.Config.Hardware.Device
	devices := object.VirtualDeviceList(device)
	disks := devices.SelectByType((*types.VirtualDisk)(nil))
	for _, d := range disks {
		vd := d.(*types.VirtualDisk)
		if *vd.UnitNumber == 0 {
			sysDisk = vc.buildDiskInfo(vmID, vd)
		} else {
			dataDisk := vc.buildDiskInfo(vmID, vd)
			dataDisks = append(dataDisks, dataDisk)
		}
	}
	return sysDisk, dataDisks
}

func (vc *VCenter) findNetworkInterfaces(moVM mo.VirtualMachine) []protocol.NetworkInterfaceInfo {
	var networkInterfaces []protocol.NetworkInterfaceInfo
	// ip
	ipMap := make(map[string][]protocol.IpInfo)
	if moVM.Guest != nil {
		for _, net := range moVM.Guest.Net {
			mac := net.MacAddress
			ipCfg := net.IpConfig
			var ipInfos []protocol.IpInfo
			if ipCfg != nil {
				ipInfo := protocol.IpInfo{}
				for _, ip := range ipCfg.IpAddress {
					ipInfo.IpAddress = ip.IpAddress
					ipInfo.State = ip.State
					ipInfos = append(ipInfos, ipInfo)
				}
			}
			ipMap[mac] = ipInfos
		}
	}

	// nic
	device := moVM.Config.Hardware.Device
	devices := object.VirtualDeviceList(device)
	nics := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	if nics != nil {
		for _, nic := range nics {
			info := vc.buildNetworkInterfaceInfo(nic)
			info.ID = vc.buildDeviceId(moVM.Reference().Value, info.Key)
			info.IPs = ipMap[info.MACAddress]

			networkInterfaces = append(networkInterfaces, info)
		}
	}
	return networkInterfaces
}

func (vc *VCenter) buildDeviceId(vmID string, key int32) string {
	return fmt.Sprintf("%s:%d", vmID, key)
}

func (vc *VCenter) buildDiskInfo(vmID string, d *types.VirtualDisk) protocol.DiskInfo {
	var diskInfo protocol.DiskInfo
	key := d.GetVirtualDevice().Key
	diskInfo.ID = vc.buildDeviceId(vmID, key)
	diskInfo.Key = key
	diskInfo.Size = int32(d.CapacityInKB / 1024 / 1024)
	format, sharing, mode, datastoreID := vc.getDiskBackingInfo(d)
	diskInfo.Format = format
	diskInfo.Mode = mode
	diskInfo.Sharing = sharing
	diskInfo.DatastoreID = datastoreID
	return diskInfo
}

func (vc *VCenter) buildNetworkInterfaceInfo(nic types.BaseVirtualDevice) protocol.NetworkInterfaceInfo {
	info := protocol.NetworkInterfaceInfo{}
	info.NetworkID = nic.GetVirtualDevice().Backing.(*types.VirtualEthernetCardNetworkBackingInfo).Network.Value
	info.Key = nic.GetVirtualDevice().Key
	switch nic.(type) {
	case *types.VirtualE1000:
		info.AdapterType = "e1000"
		info.MACAddress = nic.(*types.VirtualE1000).MacAddress
	case *types.VirtualE1000e:
		info.AdapterType = "e1000e"
		info.MACAddress = nic.(*types.VirtualE1000e).MacAddress
	case *types.VirtualPCNet32:
		info.AdapterType = "pcnet32"
		info.MACAddress = nic.(*types.VirtualPCNet32).MacAddress
	case *types.VirtualVmxnet:
		info.AdapterType = "vmxnet"
		info.MACAddress = nic.(*types.VirtualVmxnet).MacAddress
	case *types.VirtualVmxnet2:
		info.AdapterType = "vmxnet2"
		info.MACAddress = nic.(*types.VirtualVmxnet2).MacAddress
	case *types.VirtualVmxnet3:
		info.AdapterType = "vmxnet3"
		info.MACAddress = nic.(*types.VirtualVmxnet3).MacAddress
	}
	return info
}

// findDiskFormatAndMode
// return format, sharing, mode, datastoreId
func (vc *VCenter) getDiskBackingInfo(d *types.VirtualDisk) (*string, *string, *string, *string) {
	var format, sharing, mode, datastoreId string
	backing := d.Backing
	switch backing.(type) {
	case *types.VirtualDiskFlatVer2BackingInfo:
		flatVer2 := backing.(*types.VirtualDiskFlatVer2BackingInfo)
		mode = flatVer2.DiskMode
		format = disk.GetFormat(flatVer2.EagerlyScrub, flatVer2.ThinProvisioned)
		sharing = flatVer2.Sharing
		datastoreId = flatVer2.Datastore.Value
	case *types.VirtualDiskFlatVer1BackingInfo:
		flatVer1 := backing.(*types.VirtualDiskFlatVer1BackingInfo)
		mode = flatVer1.DiskMode
		datastoreId = flatVer1.Datastore.Value
	case *types.VirtualDiskRawDiskMappingVer1BackingInfo:
		rawDiskMappingVer1 := backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo)
		mode = rawDiskMappingVer1.DiskMode
		sharing = rawDiskMappingVer1.Sharing
		datastoreId = rawDiskMappingVer1.Datastore.Value
	case *types.VirtualDiskSeSparseBackingInfo:
		seSparse := backing.(*types.VirtualDiskSeSparseBackingInfo)
		mode = seSparse.DiskMode
		datastoreId = seSparse.Datastore.Value
	case *types.VirtualDiskSparseVer1BackingInfo:
		sparseVer1 := backing.(*types.VirtualDiskSparseVer1BackingInfo)
		mode = sparseVer1.DiskMode
		datastoreId = sparseVer1.Datastore.Value
	case *types.VirtualDiskSparseVer2BackingInfo:
		sparseVer2 := backing.(*types.VirtualDiskSparseVer2BackingInfo)
		mode = sparseVer2.DiskMode
		datastoreId = sparseVer2.Datastore.Value
	case *types.VirtualDiskRawDiskVer2BackingInfo:
		rawDiskVer2 := backing.(*types.VirtualDiskRawDiskVer2BackingInfo)
		sharing = rawDiskVer2.Sharing
	case *types.VirtualDiskPartitionedRawDiskVer2BackingInfo:
		partitionedRawDiskVer2 := backing.(*types.VirtualDiskPartitionedRawDiskVer2BackingInfo)
		sharing = partitionedRawDiskVer2.Sharing
	}

	return &format, &sharing, &mode, &datastoreId
}
