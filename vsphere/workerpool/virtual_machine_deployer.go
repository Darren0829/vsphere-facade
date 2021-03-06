package workerpool

import (
	"github.com/google/uuid"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"time"
	"vsphere-facade/app/logging"
	"vsphere-facade/config"
	"vsphere-facade/helper"
	"vsphere-facade/helper/datastore"
	"vsphere-facade/helper/virtualmachine"
	"vsphere-facade/helper/virtualmachine/virtualmachineclone"
	"vsphere-facade/helper/virtualmachine/virtualmachinecustomize"
	"vsphere-facade/helper/virtualmachine/virtualmachinereconfig"
)

type VirtualMachineDeployer struct {
	DeployID       string
	TimeoutSetting *TimeoutSetting
	Parameter      DeployParameter

	api       *helper.API
	oNewVM    *object.VirtualMachine
	newVmID   string
	newVmName string
}

type DeployParameter struct {
	Name              string                                   `json:"name" valid:"Required"`
	Template          Template                                 `json:"template" valid:"Required"`
	Location          virtualmachineclone.LocationParameter    `json:"location" valid:"Required"`
	Cpu               *virtualmachinereconfig.CpuParameter     `json:"cpu"`
	Memory            *virtualmachinereconfig.MemoryParameter  `json:"memory"`
	NetworkInterfaces []*NetworkInterface                      `json:"networkInterfaces,omitempty"`
	DataDisks         []*DataDisk                              `json:"dataDisks,omitempty"`
	OS                *OS                                      `json:"os,omitempty"`
	GlobalIP          *virtualmachinecustomize.GlobalIPSetting `json:"globalIp,omitempty"`
	WaitForIP         string                                   `json:"waitForIp,omitempty"`
	Flag              Flag                                     `json:"flag,omitempty"`
	PowerOn           *bool                                    `json:"powerOn,omitempty"`
}

type Template struct {
	ID        string      `json:"id"`
	SysDisk   *SysDisk    `json:"sysDisk"`
	DataDisks []*DataDisk `json:"dataDisks,omitempty"`
}

type Location struct {
	DatacenterId   string
	FolderId       string
	ClusterId      string
	HostId         string
	ResourcePoolId string
	DatastoreId    string
}

type NetworkInterface struct {
	id          string
	key         int32
	NetworkID   string                                    `json:"networkId"`
	AdapterType *string                                   `json:"adapterType,omitempty"`
	MACAddress  *string                                   `json:"macAddress,omitempty"`
	Allocation  *virtualmachinereconfig.NetworkAllocation `json:"allocation,omitempty"`

	DnsServerList []string                                `json:"dnsServerList"`
	DnsDomain     *string                                 `json:"dnsDomain"`
	Gateway       []string                                `json:"gateway"`
	SubnetMask    *int32                                  `json:"subnetMask"`
	IPv4          *virtualmachinecustomize.NicIPv4Setting `json:"ipv4,omitempty"`
	IPv6          *virtualmachinecustomize.NicIPv6Setting `json:"ipv6,omitempty"`
}

type SysDisk struct {
	DatastoreId *string `json:"datastoreId"`

	Format          *string `json:"format,omitempty"`
	Size            *int32  `json:"size,omitempty"`
	Mode            *string `json:"mode,omitempty"`
	StoragePolicyID *string `json:"storagePolicyId,omitempty"`
}

type DataDisk struct {
	DatastoreId string `json:"datastoreId"`

	Format          string  `json:"format"`
	Size            int32   `json:"size"`
	Mode            *string `json:"mode"`
	StoragePolicyID *string `json:"storagePolicyId"`
}

type Cpu struct {
	NumCPUs           int32 `json:"numCPUs"`
	NumCoresPerSocket int32 `json:"numCoresPerSocket"`

	CpuHotAddEnabled    *bool `json:"cpuHotAddEnabled"`
	CpuHotRemoveEnabled *bool `json:"cpuHotRemoveEnabled"`

	Reservation           *int64  `json:"reservation"`
	ExpandableReservation *bool   `json:"expandableReservation"`
	Limit                 *int64  `json:"limit"`
	Shares                *int32  `json:"shares"`
	Level                 *string `json:"level"`
	OverheadLimit         *int64  `json:"overheadLimit"`
}

type Memory struct {
	MemoryMB int64 `json:"memoryMB"`

	MemoryHotAddEnabled *bool `json:"memoryHotAddEnabled"`

	Reservation           *int64  `json:"reservation"`
	ExpandableReservation *bool   `json:"expandableReservation"`
	Limit                 *int64  `json:"limit"`
	Shares                *int32  `json:"shares"`
	Level                 *string `json:"level"`
	OverheadLimit         *int64  `json:"overheadLimit"`
}

type OS struct {
	Linux   *virtualmachinecustomize.LinuxSettingParameter   `json:"linux,omitempty"`
	Windows *virtualmachinecustomize.WindowsSettingParameter `json:"windows,omitempty"`
}

type TimeoutSetting struct {
	WaitForClone int32  `json:"waitForClone,omitempty"`
	WaitForIP    *int32 `json:"waitForIP,omitempty"`
	WaitForNet   *int32 `json:"waitForNet,omitempty"`
}

type Flag struct {
	EnableLogging *bool `json:"enableLogging,omitempty"`
}

func NewVirtualMachineDeployer(api *helper.API) *VirtualMachineDeployer {
	return &VirtualMachineDeployer{
		api: api,
	}
}

func (d *VirtualMachineDeployer) Deploy() error {
	d.setTimeout()
	d.setDefault()
	// ????????????
	oTempVM := virtualmachine.GetObject(d.api, d.Parameter.Template.ID)
	var clone = virtualmachineclone.CloneParameter{}
	clone.ID = d.Parameter.Template.ID
	clone.Name = d.Parameter.Name
	clone.Location = virtualmachineclone.LocationParameter{
		DatacenterID:   d.Parameter.Location.DatacenterID,
		FolderID:       d.Parameter.Location.FolderID,
		ClusterID:      d.Parameter.Location.ClusterID,
		HostId:         d.Parameter.Location.HostId,
		ResourcePoolID: d.Parameter.Location.ResourcePoolID,
		DatastoreID:    d.Parameter.Location.DatastoreID,
	}

	if d.Parameter.Template.SysDisk != nil {
		disk := virtualmachine.GetSysDisk(oTempVM)
		var sysDisk = virtualmachineclone.SelfContainedDiskParameter{}
		sysDisk.Key = disk.Key
		sysDisk.Mode = d.Parameter.Template.SysDisk.Mode
		sysDisk.Format = d.Parameter.Template.SysDisk.Format
		sysDisk.DatastoreID = d.Parameter.Template.SysDisk.DatastoreId
		sysDisk.StoragePolicyID = d.Parameter.Template.SysDisk.StoragePolicyID
		clone.Disks = &[]virtualmachineclone.SelfContainedDiskParameter{sysDisk}
	}
	oVM, err := virtualmachineclone.Clone(d.api, clone, d.TimeoutSetting.WaitForClone)
	if oVM == nil {
		logging.L().Error("?????????????????????", err)
		return err
	}
	d.setNewVM(oVM)

	// ????????????
	reconfig := virtualmachinereconfig.ReconfigureParameter{}
	createDate := time.Now()
	reconfig.CreateDate = &createDate
	cpu := d.Parameter.Cpu
	if cpu != nil {
		reconfig.Cpu = &virtualmachinereconfig.CpuParameter{
			NumCPU:              cpu.NumCPU,
			NumCoresPerSocket:   cpu.NumCoresPerSocket,
			CpuHotAddEnabled:    cpu.CpuHotAddEnabled,
			CpuHotRemoveEnabled: cpu.CpuHotRemoveEnabled,
			Allocation:          cpu.Allocation,
		}
	}
	memory := d.Parameter.Memory
	if memory != nil {
		reconfig.Memory = &virtualmachinereconfig.MemoryParameter{
			MemoryMB:            memory.MemoryMB,
			MemoryHotAddEnabled: memory.MemoryHotAddEnabled,
			Allocation:          memory.Allocation,
		}
	}
	nics := d.Parameter.NetworkInterfaces

	var nicParameter = virtualmachinereconfig.NicParameter{}
	// ????????????????????????
	cards := virtualmachine.GetEthernetCards(oVM)
	for _, card := range cards {
		nicParameter.Remove = append(nicParameter.Remove, card.GetVirtualDevice().Key)
	}
	if nics != nil {
		// ??????????????????
		for _, nic := range nics {
			nic.id = uuid.NewString()
			nicParameter.Add = append(nicParameter.Add, &virtualmachinereconfig.AddNicParameter{
				ID:          nic.id,
				NetworkID:   nic.NetworkID,
				AdapterType: nic.AdapterType,
				MACAddress:  nic.MACAddress,
				Allocation:  nic.Allocation,
			})
		}
		reconfig.Nic = &nicParameter
	}
	oVM, err = virtualmachinereconfig.Reconfigure(d.api, d.newVmID, &reconfig)
	if err != nil {
		logging.L().Error("??????????????????", err)
		d.rollBack()
		return err
	}

	// ???????????????????????????Key??????????????????????????????
	if nicParameter.Add != nil {
		newCards := virtualmachine.GetEthernetCards(oVM)
		var unKeymap = make(map[int32]int32)
		for _, card := range newCards {
			device := card.GetVirtualDevice()
			if device.UnitNumber != nil {
				un := *device.UnitNumber
				unKeymap[un] = device.Key
			}
		}

		var unMap = make(map[string]int32)
		for _, n := range nicParameter.Add {
			unMap[n.ID] = *n.UnitNumber
		}

		for _, networkInterface := range d.Parameter.NetworkInterfaces {
			id := networkInterface.id
			un := unMap[id]
			key := unKeymap[un]
			networkInterface.key = key
		}
	}

	// ????????????
	shouldCustomize := false
	customize := virtualmachinecustomize.CustomizeParameter{}
	customize.GlobalIPSetting = d.Parameter.GlobalIP
	if d.Parameter.OS != nil {
		osInfo := virtualmachine.GetOSInfo(d.api, d.oNewVM)
		customize.OSSetting = &virtualmachinecustomize.OSSettingParameter{
			LinuxSetting:   d.Parameter.OS.Linux,
			WindowsSetting: d.Parameter.OS.Windows,
		}
		if string(types.VirtualMachineGuestOsFamilyWindowsGuest) == osInfo.GuestFamily {
			if customize.OSSetting.WindowsSetting != nil {
				shouldCustomize = true
			}
		} else {
			if customize.OSSetting.LinuxSetting != nil {
				shouldCustomize = true
			}
		}
	}

	if d.Parameter.NetworkInterfaces != nil {
		for _, n := range d.Parameter.NetworkInterfaces {
			if n.IPv4 == nil && n.IPv6 == nil {
				// ????????????IP???????????????DHCP
				n.IPv4 = &virtualmachinecustomize.NicIPv4Setting{
					Static: false,
				}
			}
			nicSetting := virtualmachinecustomize.NicSetting{}
			nicSetting.Key = n.key
			nicSetting.DnsDomain = n.DnsDomain
			nicSetting.Gateway = n.Gateway
			nicSetting.SubnetMask = n.SubnetMask
			nicSetting.DnsServerList = n.DnsServerList
			nicSetting.IPv4 = n.IPv4
			nicSetting.IPv6 = n.IPv6
			customize.NicSetting = append(customize.NicSetting, &nicSetting)
		}
	}

	if shouldCustomize {
		err = virtualmachinecustomize.Customize(d.api, d.newVmID, &customize)
		if err != nil {
			logging.L().Error("", err)
			d.rollBack()
			return err
		}
	} else {
		logging.L().Debugf("????????????????????????????????????????????????")
	}

	// ??????
	err = virtualmachine.PowerOn(oVM, true, 10)
	if err != nil {
		logging.L().Error("", err)
		return err
	}

	// ??????IP
	waitForIPTimeout := d.TimeoutSetting.WaitForIP
	if waitForIPTimeout != nil && *waitForIPTimeout > 0 {
		err = virtualmachine.WaitForGuestIP(d.api, oVM, nil, d.Parameter.WaitForIP, *waitForIPTimeout)
		if err != nil {
			logging.L().Error("", err)
			d.rollBack()
			return err
		}

		waitForNetTimeout := d.TimeoutSetting.WaitForNet
		if waitForNetTimeout != nil && *waitForNetTimeout > 0 {
			err = virtualmachine.WaitForGuestNet(d.api, oVM, false, nil, *waitForNetTimeout)
			if err != nil {
				logging.L().Error("", err)
				d.rollBack()
				return err
			}
		}
	}
	return nil
}

func (d *VirtualMachineDeployer) Verify() []string {
	// todo ??????????????????
	// ??????????????????
	// ????????????????????????
	// ????????????????????????
	// ????????????????????????
	// ??????????????????????????????
	// ????????????????????????
	// ???????????????????????????
	// ???????????????????????????
	// ????????????????????????
	// ???????????????????????????????????????
	// ???????????????????????????????????????
	// ????????????????????????
	// ????????????????????????
	// ???????????????????????????????????????
	// ???????????????????????????????????????
	return nil
}

func (d *VirtualMachineDeployer) NewMachineID() string {
	if d.oNewVM == nil {
		return ""
	}
	return d.newVmID
}

func (d *VirtualMachineDeployer) setTimeout() {
	if d.TimeoutSetting == nil {
		d.TimeoutSetting = &TimeoutSetting{}
	}

	if d.TimeoutSetting.WaitForClone < 1 {
		d.TimeoutSetting.WaitForClone = config.G.Vsphere.Timeout.WaitForClone
	}

	if d.TimeoutSetting.WaitForNet == nil {
		d.TimeoutSetting.WaitForNet = &config.G.Vsphere.Timeout.WaitForNet
	}

	if d.TimeoutSetting.WaitForIP == nil {
		d.TimeoutSetting.WaitForIP = &config.G.Vsphere.Timeout.WaitForIP
	}
}

func (d *VirtualMachineDeployer) setDefault() {
	deployment := config.G.Vsphere.Default.Deployment
	p := d.Parameter
	if p.Flag.EnableLogging == nil {
		p.Flag.EnableLogging = deployment.Flag.EnableLogging
	}

	if len(p.NetworkInterfaces) > 0 {
		var nis []*NetworkInterface
		for _, ni := range p.NetworkInterfaces {
			if ni.AdapterType == nil {
				ni.AdapterType = deployment.AdapterType
			}
			nis = append(nis, ni)
		}
		p.NetworkInterfaces = nis
	}

	if p.Template.SysDisk != nil {
		sysDisk := *p.Template.SysDisk
		if sysDisk.DatastoreId != nil && sysDisk.StoragePolicyID == nil {
			moDatastore := datastore.GetMObject(d.api, *sysDisk.DatastoreId)
			if moDatastore != nil {
				datastoreType := moDatastore.Summary.Type
				storagePolicyID := deployment.StoragePolicies[d.api.ID][datastoreType]
				if storagePolicyID != "" {
					sysDisk.StoragePolicyID = &storagePolicyID
				}
			}
		}

		if sysDisk.Mode == nil {
			sysDisk.Mode = deployment.DiskMode
		}
		p.Template.SysDisk = &sysDisk
	}

	if len(p.Template.DataDisks) > 0 {
		var dataDisks []*DataDisk
		for _, disk := range p.Template.DataDisks {
			if disk.StoragePolicyID == nil {
				moDatastore := datastore.GetMObject(d.api, disk.DatastoreId)
				if moDatastore != nil {
					datastoreType := moDatastore.Summary.Type
					storagePolicyID := deployment.StoragePolicies[d.api.ID][datastoreType]
					if storagePolicyID != "" {
						disk.StoragePolicyID = &storagePolicyID
					}
				}
			}

			if disk.Mode == nil {
				disk.Mode = deployment.DiskMode
			}
			dataDisks = append(dataDisks, disk)
		}
		d.Parameter.DataDisks = dataDisks
	}
	d.Parameter = p
}

func (d *VirtualMachineDeployer) rollBack() {
	if d.oNewVM != nil {
		logging.L().Debugf("?????????????????????????????????%s(%s)", d.oNewVM.Name(), d.oNewVM.Reference().Value)
		err := virtualmachine.Destroy(d.api, d.oNewVM.Reference().Value)
		if err != nil {
			logging.L().Errorf("?????????????????????????????????%s(%s)????????????: %v", d.oNewVM.Name(), d.oNewVM.Reference().Value, err)
			return
		}
	}
}

func (d *VirtualMachineDeployer) setNewVM(oVM *object.VirtualMachine) {
	d.oNewVM = oVM
	d.newVmID = oVM.Reference().Value
	d.newVmName = oVM.Name()
}
