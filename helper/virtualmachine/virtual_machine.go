package virtualmachine

import (
	"context"
	"errors"
	"fmt"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"net"
	"strings"
	"time"
	"vsphere_api/app/logging"
	"vsphere_api/helper"
	"vsphere_api/helper/clustercomputerresource"
	"vsphere_api/helper/computerresource"
	"vsphere_api/helper/datacenter"
	"vsphere_api/helper/folder"
	"vsphere_api/helper/hostsystem"
	"vsphere_api/helper/vsphere"
)

const Type = "VirtualMachine"

type retrieveType int

const (
	RetrieveMachineOnly retrieveType = iota
	RetrieveTemplateOnly
	RetrieveAll
)

var templateProps = []string{
	"config.name",
	"config.uuid",
	"config.instanceUuid",
	"config.hardware.numCPU",
	"config.hardware.numCoresPerSocket",
	"config.hardware.memoryMB",
	"config.hardware.device",
	"config.template",
	"config.annotation",
	"config.tools.toolsInstallType",
	"config.guestId",
	"config.guestFullName",
	"runtime.host",
	"resourcePool",
}

var virtualMachineProps = []string{
	"name",
	"config.name",
	"config.uuid",
	"config.instanceUuid",
	"config.hardware.numCPU",
	"config.hardware.numCoresPerSocket",
	"config.hardware.memoryMB",
	"config.hardware.device",
	"config.template",
	"config.annotation",
	"config.tools.toolsInstallType",
	"config.guestId",
	"config.guestFullName",
	"runtime.host",
	"runtime.powerState",
	"resourcePool",
	"guest",
}

type OSInfo struct {
	GuestID     string
	GuestName   string
	GuestFamily string
}

type ConfigureInfo struct {
	NumCPU            int32
	NumCoresPerSocket int32
	MemoryMB          int32
}

type LocationInfo struct {
	DatacenterID   string
	ResourcePoolID string
	ClusterID      string
	HostID         string
	FolderID       string
}

func GetObject(api *helper.API, ID string) *object.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取虚拟机", ID))
	finder := find.NewFinder(api.Client.Client, false)

	ref := types.ManagedObjectReference{
		Type:  Type,
		Value: ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	or, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		logging.L().Error(fmt.Sprintf("使用ID[%s]获取虚拟机时,发生错误", ID), err)
		return nil
	}

	return or.(*object.VirtualMachine)
}

func GetMObject(api *helper.API, ID string) *mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("使用ID[%s]获取虚拟机", ID))
	oVM := GetObject(api, ID)
	if oVM == nil {
		return nil
	}
	return FindAllProps(oVM)
}

func FindAllProps(oVM *object.VirtualMachine) *mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的所有属性", oVM.Name(), oVM.Reference().Value))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moVM mo.VirtualMachine
	if err := oVM.Properties(ctx, oVM.Reference(), nil, &moVM); err != nil {
		return nil
	}
	return &moVM
}

func FindProps(oVM *object.VirtualMachine, props ...string) *mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的属性%s", oVM.Name(), oVM.Reference().Value, props))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	var moVM mo.VirtualMachine
	if err := oVM.Properties(ctx, oVM.Reference(), props, &moVM); err != nil {
		return nil
	}
	return &moVM
}

func GetDevices(oVM *object.VirtualMachine) object.VirtualDeviceList {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的设备列表", oVM.Name(), oVM.Reference().Value))
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	devices, err := oVM.Device(ctx)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取虚拟机[%s(%s)]设备时发生错误", oVM.Name(), oVM.Reference().Value), err)
		return nil
	}
	return devices
}

func GetSysDisk(oVM *object.VirtualMachine) *types.VirtualDisk {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的系统盘信息", oVM.Name(), oVM.Reference().Value))
	disks := GetDisks(oVM)
	for _, disk := range disks {
		d := disk.GetVirtualDevice()
		if *d.UnitNumber == 0 {
			return disk.(*types.VirtualDisk)
		}
	}
	return nil
}

func GetDisks(oVM *object.VirtualMachine) object.VirtualDeviceList {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的磁盘列表", oVM.Name(), oVM.Reference().Value))
	devices := GetDevices(oVM)
	return devices.SelectByType((*types.VirtualDisk)(nil))
}

func GetEthernetCards(oVM *object.VirtualMachine) object.VirtualDeviceList {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的网卡列表", oVM.Name(), oVM.Reference().Value))
	devices := GetDevices(oVM)
	return devices.SelectByType((*types.VirtualEthernetCard)(nil))
}

func GetOSInfo(api *helper.API, oVM *object.VirtualMachine) OSInfo {
	logging.L().Debug(fmt.Sprintf("获取虚拟机[%s(%s)]的OS信息", oVM.Name(), oVM.Reference().Value))
	props := FindProps(oVM, "guest", "config", "runtime")
	guest := props.Guest
	if guest.GuestId != "" {
		return OSInfo{
			GuestID:     guest.GuestId,
			GuestName:   guest.GuestFullName,
			GuestFamily: guest.GuestFamily,
		}
	} else {
		config := props.Config
		hostID := props.Runtime.Host.Reference().Value
		computerResourceID := hostsystem.GetMObject(api, hostID).Parent.Value
		OSFamilies := computerresource.GetOSFamilies(api, computerResourceID)
		for _, family := range OSFamilies {
			if family.Id == config.GuestId {
				return OSInfo{
					GuestID:     family.Id,
					GuestName:   family.FullName,
					GuestFamily: family.Family,
				}
			}
		}
		return OSInfo{}
	}
}

func GetVirtualMachinesByDatacenterID(api *helper.API, datacenterID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取数据中心[%s]下的所有虚拟机", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveMachineOnly, virtualMachineProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取数据中心[%s]下的所有虚拟机时发生错误", datacenterID), err)
		return nil
	}
	return templates
}

func GetVirtualMachinesByClusterID(api *helper.API, clusterID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取集群[%s]下的所有虚拟机", clusterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  computerresource.Type,
		Value: clusterID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveMachineOnly, virtualMachineProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取集群[%s]下的所有虚拟机时发生错误", clusterID), err)
		return nil
	}
	return templates
}

func GetVirtualMachinesByHostID(api *helper.API, hostID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取主机[%s]下的所有虚拟机", hostID))
	moDatacenter := types.ManagedObjectReference{
		Type:  hostsystem.Type,
		Value: hostID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveMachineOnly, virtualMachineProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取主机[%s]下的所有虚拟机时发生错误", hostID), err)
		return nil
	}
	return templates
}

func GetVirtualMachinesByFolderID(api *helper.API, folderID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("查询文件夹[%s]下的所有主机", folderID))
	moDatacenter := types.ManagedObjectReference{
		Type:  folder.Type,
		Value: folderID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveMachineOnly, templateProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询文件夹[%s]下的所有虚拟机时发生错误", folderID), err)
		return nil
	}
	return templates
}

func GetTemplatesByDatacenterID(api *helper.API, datacenterID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("查询数据中心[%s]下的所有模版", datacenterID))
	moDatacenter := types.ManagedObjectReference{
		Type:  datacenter.Type,
		Value: datacenterID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveTemplateOnly, templateProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("查询数据中心[%s]下的所有模版时发生错误", datacenterID), err)
		return nil
	}
	return templates
}

func GetTemplatesByFolderID(api *helper.API, folderID string) []mo.VirtualMachine {
	logging.L().Debug(fmt.Sprintf("获取文件夹[%s]下的所有模版", folderID))
	moDatacenter := types.ManagedObjectReference{
		Type:  folder.Type,
		Value: folderID,
	}

	templates, err := retrieve(api, moDatacenter, RetrieveTemplateOnly, templateProps...)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取文件夹[%s]下的所有模版时发生错误", folderID), err)
		return nil
	}
	return templates
}

func PowerOn(oVM *object.VirtualMachine, ignoreFailedState bool, timeout int32) error {
	logging.L().Debug(fmt.Sprintf("开启虚拟机[%s(%s)]电源", oVM.Name(), oVM.Reference().Value))
	if ignoreFailedState {
		if GetPowerState(oVM) == types.VirtualMachinePowerStatePoweredOn {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	oTask, err := oVM.PowerOn(ctx)
	if err != nil {
		return err
	}

	if timeout > 0 {
		tctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
		defer cancel()
		oTaskInfo, err := oTask.WaitForResult(tctx, nil)
		if err != nil {
			return err
		}
		if oTaskInfo.Error != nil {
			return errors.New(oTaskInfo.Error.LocalizedMessage)
		}
	}
	return nil
}

func PowerOff(oVM *object.VirtualMachine, ignoreFailedState bool, timeout int32) error {
	logging.L().Debug(fmt.Sprintf("关闭虚拟机[%s(%s)]电源", oVM.Name(), oVM.Reference().Value))
	if ignoreFailedState {
		if GetPowerState(oVM) == types.VirtualMachinePowerStatePoweredOff {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	oTask, err := oVM.PowerOff(ctx)
	if err != nil {
		return err
	}

	if timeout > 0 {
		tctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
		defer cancel()
		oTaskInfo, err := oTask.WaitForResult(tctx, nil)
		if err != nil {
			return err
		}
		if oTaskInfo.Error != nil {
			return errors.New(oTaskInfo.Error.LocalizedMessage)
		}
	}
	return nil
}

func Shutdown(oVM *object.VirtualMachine, ignoreFailedState bool, timeout int32) error {
	logging.L().Debug(fmt.Sprintf("关闭虚拟机[%s(%s)]操作系统", oVM.Name(), oVM.Reference().Value))
	if ignoreFailedState {
		if GetPowerState(oVM) == types.VirtualMachinePowerStatePoweredOff {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	err := oVM.ShutdownGuest(ctx)
	if err != nil {
		return err
	}

	if timeout > 0 {
		tctx, tcancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
		defer tcancel()
		err = oVM.WaitForPowerState(tctx, types.VirtualMachinePowerStatePoweredOff)
		if err != nil {
			return err
		}
	}
	return nil
}

func Destroy(api *helper.API, ID string) error {
	logging.L().Debug(fmt.Sprintf("删除虚拟机[%s]", ID))
	oVM := GetObject(api, ID)
	if oVM == nil {
		return nil
	}

	if GetPowerState(oVM) != types.VirtualMachinePowerStatePoweredOff {
		err := PowerOff(oVM, true, 10)
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	oTask, err := oVM.Destroy(ctx)
	if err != nil {
		return err
	}
	return vsphere.WaitForTask(oTask, 2)
}

func Rename(oVM *object.VirtualMachine, newName string, timeout int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	oTask, err := oVM.Rename(ctx, newName)
	if err != nil {
		return err
	}
	if timeout > 0 {
		return vsphere.WaitForTask(oTask, timeout)
	}
	return nil
}

func GetConfigureInfo(oVM *object.VirtualMachine) ConfigureInfo {
	props := FindProps(oVM, "config.hardware.memoryMB",
		"config.hardware.numCPU",
		"config.hardware.numCoresPerSocket")
	return ConfigureInfo{
		NumCPU:            props.Config.Hardware.NumCPU,
		NumCoresPerSocket: props.Config.Hardware.NumCoresPerSocket,
		MemoryMB:          props.Config.Hardware.MemoryMB,
	}
}

func GetGuestNicInfo(oVM *object.VirtualMachine) []types.GuestNicInfo {
	props := FindProps(oVM, "guest.net")
	return props.Guest.Net
}

func GetHostname(oVM *object.VirtualMachine) string {
	props := FindProps(oVM, "guest.hostName")
	return props.Guest.HostName
}

func IsLinux(api *helper.API, oVM *object.VirtualMachine) bool {
	props := FindProps(oVM, "config.guestId")
	guestId := props.Config.GuestId

	location := GetLocationInfo(api, oVM)
	var os types.GuestOsDescriptor
	if location.ClusterID != "" {
		os = *computerresource.GetOSFamily(api, location.ClusterID, guestId)
	} else {
		os = *computerresource.GetOSFamily(api, location.HostID, guestId)
	}
	return os.Family == "linuxGuest"
}

func GetLocationInfo(api *helper.API, oVM *object.VirtualMachine) *LocationInfo {
	location := LocationInfo{}
	props := FindProps(oVM, "runtime", "parent", "parentVApp", "resourcePool")
	host := props.Runtime.Host
	location.HostID = host.Reference().Value

	moHost := hostsystem.GetMObject(api, location.HostID)
	hostParent := moHost.Parent
	if hostParent.Type == clustercomputerresource.Type {
		moCluster := clustercomputerresource.GetMObject(api, hostParent.Value)
		location.ClusterID = moCluster.Reference().Value
	}
	if props.ResourcePool != nil {
		location.ResourcePoolID = props.ResourcePool.Value
	}

	parentPath := vsphere.FindParentPathByType(api, oVM.Reference().Value, Type, datacenter.Type)
	if parentPath != nil {
		plen := len(parentPath)
		location.DatacenterID = parentPath[plen-1].Value
		if plen > 2 {
			location.FolderID = parentPath[0].Value
		}
	}
	return &location
}

func GetPowerState(oVM *object.VirtualMachine) types.VirtualMachinePowerState {
	props := FindProps(oVM, "runtime.powerState")
	return props.Runtime.PowerState
}

func WaitForGuestIP(api *helper.API, oVM *object.VirtualMachine, ignoreIPs []interface{}, targetIP string, timeout int32) error {
	logging.L().Debugf("等待虚拟机[%s(%s)]IP生效", oVM.Name(), oVM.Reference().Value)
	if targetIP != "" {
		logging.L().Debugf("等待虚拟机[%s(%s)]IP变为: %s", oVM.Name(), oVM.Reference().Value, targetIP)
	}
	p := api.Client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
	defer cancel()
	err := property.Wait(ctx, p, oVM.Reference(), []string{"guest.ipAddress"}, func(pc []types.PropertyChange) bool {
		for _, c := range pc {
			if c.Op != types.PropertyChangeOpAssign {
				continue
			}
			if c.Val == nil {
				continue
			}
			ip := net.ParseIP(c.Val.(string))
			logging.L().Debugf("获取到Guest IP: %s", ip.String())
			if skipIPAddrForWaiter(ip, ignoreIPs) {
				continue
			}
			if targetIP != "" && !isSpecificIP(ip, targetIP) {
				logging.L().Debugf("等待Guest IP变为: %s", targetIP)
				continue
			}
			return true
		}
		return false
	})

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("等待IP超时")
		}
		return err
	}

	return nil
}

func WaitForGuestNet(api *helper.API, oVM *object.VirtualMachine, routable bool, ignoredIPs []interface{}, timeout int32) error {
	logging.L().Debug(fmt.Sprintf("等待虚拟机[%s(%s)]网络", oVM.Name(), oVM.Reference().Value))
	var v4gw, v6gw net.IP
	p := api.Client.PropertyCollector()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
	defer cancel()

	err := property.Wait(ctx, p, oVM.Reference(), []string{"guest.net", "guest.ipStack"}, func(pc []types.PropertyChange) bool {
		for _, c := range pc {
			if c.Op != types.PropertyChangeOpAssign {
				continue
			}

			switch v := c.Val.(type) {
			case types.ArrayOfGuestStackInfo:
				for _, s := range v.GuestStackInfo {
					if s.IpRouteConfig != nil {
						for _, r := range s.IpRouteConfig.IpRoute {
							switch r.Network {
							case "0.0.0.0":
								v4gw = net.ParseIP(r.Gateway.IpAddress)
							case "::":
								v6gw = net.ParseIP(r.Gateway.IpAddress)
							}
						}
					}
				}
			case types.ArrayOfGuestNicInfo:
				for _, n := range v.GuestNicInfo {
					if n.IpConfig != nil {
						for _, addr := range n.IpConfig.IpAddress {
							ip := net.ParseIP(addr.IpAddress)
							if skipIPAddrForWaiter(ip, ignoredIPs) {
								continue
							}
							if !routable {
								// We are done. The rest of this block concerns itself with
								// checking for a routable address, but the waiter has been
								// flagged to not wait for one.
								return true
							}
							var mask net.IPMask
							if ip.To4() != nil {
								mask = net.CIDRMask(int(addr.PrefixLength), 32)
							} else {
								mask = net.CIDRMask(int(addr.PrefixLength), 128)
							}
							if ip.Mask(mask).Equal(v4gw.Mask(mask)) || ip.Mask(mask).Equal(v6gw.Mask(mask)) {
								return true
							}
						}
					}
				}
			}
		}
		return false
	})

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("等待IP超时")
		}
		return err
	}
	return nil
}

func skipIPAddrForWaiter(ip net.IP, ignoredIPs []interface{}) bool {
	switch {
	case ip.IsLinkLocalMulticast():
		fallthrough
	case ip.IsLinkLocalUnicast():
		fallthrough
	case ip.IsLoopback():
		fallthrough
	case ip.IsMulticast():
		return true
	default:
		// ignoredIPs prevalidated by Schema!
		for _, ignoredIP := range ignoredIPs {
			if strings.Contains(ignoredIP.(string), "/") {
				_, ignoredIPNet, _ := net.ParseCIDR(ignoredIP.(string))
				if ignoredIPNet.Contains(ip) {
					return true
				}
			} else if net.ParseIP(ignoredIP.(string)).Equal(ip) {
				return true
			}
		}
	}
	return false
}

func isSpecificIP(ip net.IP, specificIP string) bool {
	switch {
	case ip.IsLinkLocalMulticast():
		fallthrough
	case ip.IsLinkLocalUnicast():
		fallthrough
	case ip.IsLoopback():
		fallthrough
	case ip.IsMulticast():
		return true
	default:
		if strings.Contains(specificIP, "/") {
			_, ignoredIPNet, _ := net.ParseCIDR(specificIP)
			if ignoredIPNet.Contains(ip) {
				return true
			}
		} else if net.ParseIP(specificIP).Equal(ip) {
			return true
		}
	}
	return false
}

func retrieve(api *helper.API, mor types.ManagedObjectReference, t retrieveType, props ...string) ([]mo.VirtualMachine, error) {
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

	var moVirtualMachines []mo.VirtualMachine
	err = v.Retrieve(vctx, []string{Type}, props, &moVirtualMachines)
	if err != nil {
		return nil, err
	}

	var newVirtualMachines []mo.VirtualMachine
	switch t {
	case RetrieveMachineOnly, RetrieveTemplateOnly:
		for _, machine := range moVirtualMachines {
			if t == RetrieveTemplateOnly {
				if machine.Config.Template {
					newVirtualMachines = append(newVirtualMachines, machine)
				}
			} else if t == RetrieveMachineOnly {
				if !machine.Config.Template {
					newVirtualMachines = append(newVirtualMachines, machine)
				}
			} else {
				newVirtualMachines = append(newVirtualMachines, machine)
			}
		}
	}
	return newVirtualMachines, nil
}
