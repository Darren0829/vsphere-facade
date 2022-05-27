package vsphere

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"testing"
	"time"
	"vsphere-facade/app/cache"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/config"
	"vsphere-facade/db/badgerdb"
	"vsphere-facade/helper"
	"vsphere-facade/helper/disk"
	"vsphere-facade/helper/virtualmachine"
	"vsphere-facade/helper/virtualmachine/virtualmachineclone"
	"vsphere-facade/helper/virtualmachine/virtualmachinecustomize"
	"vsphere-facade/helper/virtualmachine/virtualmachinereconfig"
	"vsphere-facade/helper/virtualmachine/virtualmachinerelocate"
	vCache "vsphere-facade/vsphere/cache"
	"vsphere-facade/vsphere/protocol"
	"vsphere-facade/vsphere/workerpool"
)

var vc35 = Auth{
	Address:  "https://192.168.25.35",
	Username: "administrator@vsphere.locall",
	Password: "Zhu@88jie",
}

var vc30 = Auth{
	Address:  "https://192.168.25.30",
	Username: "administrator@vsphere.local",
	Password: "1qaz@WSX",
}

var vcys = Auth{
	Address:  "https://hosting51.3322.org:31444",
	Username: "administrator@leaptocloud.com",
	Password: "1qaz@WSX",
}

var vc *VCenter

func init() {
	config.Setup()
	logging.Setup()
	helper.Setup()
	cache.Setup()
	vCache.Setup()
	vc = Get(vc30)
}

func TestVCenter_QueryDatacenters(t *testing.T) {
	l := vc.QueryDatacenters(protocol.DatacenterQuery{})
	fmt.Println(utils.ToJson(l))
}

func TestVCenter_QueryClusters(t *testing.T) {
	clusters := vc.QueryClusters(protocol.ClusterQuery{DatacenterID: ""})
	fmt.Println(utils.ToJson(clusters))
}

func TestVCenter_QueryHosts(t *testing.T) {
	hosts := vc.QueryHosts(protocol.HostQuery{})
	fmt.Println(hosts)
}

func TestVCenter_QueryNetworks(t *testing.T) {
	networks := vc.QueryNetworks(protocol.NetworkQuery{})
	fmt.Println(networks)
}

func TestVCenter_QueryDatastores(t *testing.T) {
	datastores := vc.QueryDatastores(protocol.DatastoreQuery{})
	fmt.Println(utils.ToJson(datastores))
}

func TestVCenter_QueryResourcePools(t *testing.T) {
	q := protocol.ResourcePoolQuery{
		DatacenterID: "",
		ClusterID:    "",
		HostID:       "",
		IDs:          nil,
	}
	list := vc.QueryResourcePools(q)
	if list != nil {
		for _, info := range list {
			fmt.Println(info)
		}
	}
}

func TestVCenter_QueryStoragePolicies(t *testing.T) {
	l := vc.QueryStoragePolicies(protocol.StoragePolicyQuery{})
	fmt.Println(utils.ToJson(l))
}

func TestDeployVM(t *testing.T) {
	datacenterId := "datacenter-2"
	folderId := "group-v557"
	datastoreId := "datastore-16"
	resourcePoolId := "resgroup-8"
	hostId := "host-12"
	p := virtualmachineclone.CloneParameter{}
	p.ID = "vm-641"
	p.Name = "api-test"
	p.Location = virtualmachineclone.LocationParameter{
		DatacenterID:   datacenterId,
		FolderID:       &folderId,
		ClusterID:      nil,
		HostId:         &hostId,
		ResourcePoolID: &resourcePoolId,
		DatastoreID:    &datastoreId,
	}
	oVM, err := virtualmachineclone.Clone(vc.Api, p, 0)
	if oVM == nil {
		logging.L().Error("", err)
		return
	}

	logging.L().Info("开始配置虚拟机")
	props := virtualmachine.FindProps(oVM, "config")
	devices := object.VirtualDeviceList(props.Config.Hardware.Device)

	memoryMB := int32(4096)
	numCPUs := int32(2)
	numCoresPerSocket := int32(1)
	var rp = virtualmachinereconfig.ReconfigureParameter{}
	now := time.Now()
	rp.CreateDate = &now
	rp.Cpu = &virtualmachinereconfig.CpuParameter{
		NumCPU:            &numCPUs,
		NumCoresPerSocket: &numCoresPerSocket,
	}
	rp.Memory = &virtualmachinereconfig.MemoryParameter{
		MemoryMB: &memoryMB,
	}

	// 移除原有的网卡
	ethernetCardDevices := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	var nicParameter = virtualmachinereconfig.NicParameter{}
	for _, ecd := range ethernetCardDevices {
		nicParameter.Remove = append(nicParameter.Remove, ecd.GetVirtualDevice().Key)
	}

	// 添加新网卡
	AdapterType := "e1000"
	AddressType := "manual"
	MACAddress := "00:50:56:9c:1a:40"
	nicParameter.Add = append(nicParameter.Add, &virtualmachinereconfig.AddNicParameter{
		NetworkID:   "network-18",
		AdapterType: &AdapterType,
		AddressType: &AddressType,
		MACAddress:  &MACAddress,
	})
	rp.Nic = &nicParameter

	oVM, err = virtualmachinereconfig.Reconfigure(vc.Api, oVM.Reference().Value, &rp)
	if err != nil {
		logging.L().Error("重新配置虚拟机失败", err)
		return
	}

	// 自定义配置
	var c virtualmachinecustomize.CustomizeParameter
	// 网卡配置
	ethernetCards := virtualmachine.GetEthernetCards(oVM)
	var nicSetting = virtualmachinecustomize.NicSetting{}
	nicSetting.Key = ethernetCards[0].GetVirtualDevice().Key
	dnsServerList := []string{"8.8.8.8"}
	nicSetting.DnsServerList = dnsServerList
	gateway := []string{"192.168.25.1"}
	nicSetting.Gateway = gateway
	IPv4Address := "192.168.25.201"
	nicSetting.IPv4 = &virtualmachinecustomize.NicIPv4Setting{
		Static:    true,
		IPAddress: &IPv4Address,
	}
	subnetMask := int32(24)
	nicSetting.SubnetMask = &subnetMask
	c.NicSetting = []*virtualmachinecustomize.NicSetting{&nicSetting}
	// 系统信息
	hostname := "api-test"
	domain := "zhehang.deng"
	c.OSSetting = &virtualmachinecustomize.OSSettingParameter{
		LinuxSetting: &virtualmachinecustomize.LinuxSettingParameter{
			HostName: &hostname,
			Domain:   &domain,
		},
	}
	err = virtualmachinecustomize.Customize(vc.Api, oVM.Reference().Value, &c)
	if err != nil {
		logging.L().Error("虚拟机自定义配置失败", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(20))
	defer cancel()
	props = virtualmachine.FindProps(oVM, "runtime")
	if props.Runtime.PowerState != types.VirtualMachinePowerStatePoweredOn {
		powerOnTask, err := oVM.PowerOn(ctx)
		if err != nil {
			logging.L().Error("开机失败", err)
		}

		err = powerOnTask.Wait(ctx)
		if err != nil {
			logging.L().Error("等待开机失败", err)
		}
	}
}

func TestReconfigure(t *testing.T) {
	ID := "vm-1546"
	oVM := virtualmachine.GetObject(vc.Api, ID)
	logging.L().Info("开始配置虚拟机")
	props := virtualmachine.FindProps(oVM, "config")
	devices := object.VirtualDeviceList(props.Config.Hardware.Device)

	memoryMB := int32(4096)
	numCPUs := int32(2)
	numCoresPerSocket := int32(1)
	var rp = virtualmachinereconfig.ReconfigureParameter{}
	now := time.Now()
	rp.CreateDate = &now
	rp.Cpu = &virtualmachinereconfig.CpuParameter{
		NumCPU:            &numCPUs,
		NumCoresPerSocket: &numCoresPerSocket,
	}
	rp.Memory = &virtualmachinereconfig.MemoryParameter{
		MemoryMB: &memoryMB,
	}

	// 移除原有的网卡
	ethernetCardDevices := devices.SelectByType((*types.VirtualEthernetCard)(nil))
	var nicParameter = virtualmachinereconfig.NicParameter{}
	for _, ecd := range ethernetCardDevices {
		nicParameter.Remove = append(nicParameter.Remove, ecd.GetVirtualDevice().Key)
	}

	// 添加新网卡
	AdapterType := "e1000"
	AddressType := "manual"
	MACAddress := "00:50:56:9c:1a:40"
	nicParameter.Add = append(nicParameter.Add, &virtualmachinereconfig.AddNicParameter{
		NetworkID:   "network-18",
		AdapterType: &AdapterType,
		AddressType: &AddressType,
		MACAddress:  &MACAddress,
	})
	//rp.Nic = &nicParameter

	var diskParameter = virtualmachinereconfig.DiskParameter{}
	// 移除硬盘
	diskParameter.Remove = append(diskParameter.Remove, 2001)
	// 修改硬盘
	newSize := int32(35)
	storagePolicyID := "f4e5bade-15a2-4805-bf8e-52318c4ce443"
	mode := disk.ModeIndependentNonpersistent
	sharing := disk.SharingNone
	diskParameter.Edit = append(diskParameter.Edit, virtualmachinereconfig.EditDiskParameter{
		Key:             2002,
		Size:            &newSize,
		Mode:            &mode,
		StoragePolicyID: &storagePolicyID,
		Sharing:         &sharing,
	})
	// 添加硬盘
	size := int32(40)
	//storagePolicyID2 := "f4e5bade-15a2-4805-bf8e-52318c4ce443"
	mode2 := disk.ModeIndependentNonpersistent
	sharing2 := disk.SharingMultiWriter
	diskParameter.Add = append(diskParameter.Add, &virtualmachinereconfig.AddDiskParameter{
		Size:   size,
		Format: disk.FormatThin,
		Mode:   &mode2,
		//StoragePolicyID: &storagePolicyID2,
		Sharing: &sharing2,
	})
	rp.Disk = &diskParameter

	oVM, err := virtualmachinereconfig.Reconfigure(vc.Api, oVM.Reference().Value, &rp)
	if err != nil {
		logging.L().Error("重新配置虚拟机失败", err)
		return
	}
}

func TestCustomize(t *testing.T) {
	ID := "vm-1582"
	// 自定义配置
	var c virtualmachinecustomize.CustomizeParameter
	// 网卡配置
	oVM := virtualmachine.GetObject(vc.Api, ID)
	ethernetCards := virtualmachine.GetEthernetCards(oVM)
	var nicSetting = virtualmachinecustomize.NicSetting{}
	nicSetting.Key = ethernetCards[0].GetVirtualDevice().Key
	dnsServerList := []string{"8.8.8.8"}
	nicSetting.DnsServerList = dnsServerList
	gateway := []string{"192.168.25.1"}
	nicSetting.Gateway = gateway
	IPv4Address := "192.168.25.201"
	nicSetting.IPv4 = &virtualmachinecustomize.NicIPv4Setting{
		Static:    true,
		IPAddress: &IPv4Address,
	}
	subnetMask := int32(24)
	nicSetting.SubnetMask = &subnetMask
	c.NicSetting = []*virtualmachinecustomize.NicSetting{&nicSetting}
	// 系统信息
	hostname := "api-test"
	domain := "zhehang.deng"
	c.OSSetting = &virtualmachinecustomize.OSSettingParameter{
		LinuxSetting: &virtualmachinecustomize.LinuxSettingParameter{
			HostName: &hostname,
			Domain:   &domain,
		},
	}
	err := virtualmachinecustomize.Customize(vc.Api, ID, &c)
	if err != nil {
		logging.L().Error("", err)
	}
}

func TestPowerOn(t *testing.T) {
	oVM := virtualmachine.GetObject(vc.Api, "vm-1545")
	virtualmachine.PowerOn(oVM, true, 2)
}

func TestPowerOff(t *testing.T) {
	oVM := virtualmachine.GetObject(vc.Api, "vm-684")
	virtualmachine.PowerOff(oVM, false, 2)
}

func TestShutdown(t *testing.T) {
	oVM := virtualmachine.GetObject(vc.Api, "vm-1545")
	virtualmachine.Shutdown(oVM, true, 2)
}

func TestDestroy(t *testing.T) {
	virtualmachine.Destroy(vc.Api, "vm-639")
}

func TestVCenter_QueryTemplates(t *testing.T) {
	list := vc.QueryTemplates(protocol.TemplateQuery{})
	fmt.Println(utils.ToJson(list))
}

func TestVCenter_QueryFolders(t *testing.T) {
	l := vc.QueryFolders(protocol.FolderQuery{})
	fmt.Println(utils.ToJson(l))
}

func TestGetFormat(t *testing.T) {
	eager := true
	thin := false
	fmt.Println(disk.GetFormat(&eager, &thin))
}

func TestGetGetLocationInfo(t *testing.T) {
	l := virtualmachine.GetLocationInfo(vc.Api, virtualmachine.GetObject(vc.Api, "vm-641"))
	fmt.Println(utils.ToJson(l))
}

func TestUtil(t *testing.T) {
	q := protocol.CallbackReq{}
	h := utils.NilNext(q.HttpPost, config.G.Vsphere.Default.Callback.HttpPost)
	g := h.(*protocol.Http)
	fmt.Println(g.URL)
}

func TestWorkPool(t *testing.T) {
	//p := vc.GetWorkPool(workerpool.WorkerTypeOperation)
	//for i := 0; i < 20; i++ {
	//	nd := i
	//	err := p.Submit(func() {
	//		time.Sleep(time.Second * 2)
	//		logging.L().Warn("Test: ", nd)
	//	})
	//	if err != nil {
	//		logging.L().Error("", err)
	//		return
	//	}
	//}
	//logging.L().Warn("============================================")

	for i := 0; i < 200; i++ {
		nd := i
		err := workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
			time.Sleep(time.Millisecond * 500)
			logging.L().Warn("Test: ", nd)
		})
		if err != nil {
			logging.L().Error("", err)
		}
	}
	logging.L().Warn("============================================")
	time.Sleep(time.Second * 300)
}

func TestRelocate(t *testing.T) {
	p := virtualmachinerelocate.RelocateParameter{}

	//vapp := "resgroup-v1567"
	//res := "resgroup-1381"
	//host := "host-15"
	//p.Compute = &virtualmachinerelocate.ComputeParameter{
	//	DestinationID: &host,
	//}

	datastoreID := "datastore-17" //datastore-16
	//format := "thin"
	//storagePolicyID := "f4e5bade-15a2-4805-bf8e-52318c4ce443"
	//diskStorageParameter2000 := virtualmachinerelocate.DiskStorageParameter{
	//	Key:         2000,
	//	DatastoreID: "datastore-16",
	//	Format:      "thin",
	//}
	//diskStorageParameter2001 := virtualmachinerelocate.DiskStorageParameter{
	//	Key:         2001,
	//	DatastoreID: "datastore-16",
	//	Format:      "thin",
	//}
	diskStorageParameter2002 := virtualmachinerelocate.DiskStorageParameter{
		Key:         2002,
		DatastoreID: &datastoreID,
		//Format:      &format,
	}
	var diskStorageParameters []virtualmachinerelocate.DiskStorageParameter
	//diskStorageParameters = append(diskStorageParameters, diskStorageParameter2001)
	//diskStorageParameters = append(diskStorageParameters, diskStorageParameter2002)
	diskStorageParameters = append(diskStorageParameters, diskStorageParameter2002)

	p.Storage = &virtualmachinerelocate.StorageParameter{
		//StoragePolicyID:              &storagePolicyID,
		//DatastoreID:                  &datastoreID,
		Disks: diskStorageParameters,
	}

	_, err := virtualmachinerelocate.Relocate(vc.Api, "", p, 20)
	if err != nil {
		logging.L().Error("", err)
	}
}

func TestReconfigureNic(t *testing.T) {
	ID := "vm-1570"
	oVM := virtualmachine.GetObject(vc.Api, ID)

	var rp = virtualmachinereconfig.ReconfigureParameter{}

	editNicParameter := virtualmachinereconfig.EditNicParameter{}
	editNicParameter.Key = 4001
	networkID := "network-19"
	editNicParameter.NetworkID = &networkID
	mac := "00:50:56:9c:34:66"
	editNicParameter.MACAddress = &mac
	rp.Nic = &virtualmachinereconfig.NicParameter{
		Edit: []virtualmachinereconfig.EditNicParameter{editNicParameter},
	}
	oVM, err := virtualmachinereconfig.Reconfigure(vc.Api, oVM.Reference().Value, &rp)
	if err != nil {
		logging.L().Error("重新配置虚拟机失败", err)
		return
	}
}

func TestIsLinux(t *testing.T) {
	oVM := virtualmachine.GetObject(vc.Api, "vm-1570")
	b := virtualmachine.IsLinux(vc.Api, oVM)
	fmt.Println(b)
}

func TestBadgerDB_Set(t *testing.T) {
	badgerdb.Setup()
	badgerdb.Set("zhang", "san")
	badgerdb.Set("li", "si")
}

func TestBadgerDB_Get(t *testing.T) {
	badgerdb.Setup()
	v := badgerdb.Get("deng")
	fmt.Println(v)

	badgerdb.TableInfo()
}

func TestBadgerDB_Del(t *testing.T) {
	badgerdb.Setup()
	err := badgerdb.Del("li")
	if err != nil {
		fmt.Println(err)
		return
	}

	badgerdb.TableInfo()
}

func TestBadgerDB_GetAll(t *testing.T) {
	badgerdb.Setup()
	all := badgerdb.GetAll()
	for k, v := range all {
		fmt.Println(k, ":", v)
	}
}
