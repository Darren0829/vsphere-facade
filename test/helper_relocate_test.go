package test

import (
	"testing"
	"vsphere-facade/app/logging"
	"vsphere-facade/helper/virtualmachine/virtualmachinerelocate"
)

func Test_Relocate_ResourcePool(t *testing.T) {
	VMID := "vm-1622"
	p := virtualmachinerelocate.RelocateParameter{}
	destinationID := "resgroup-1381"
	p.Compute = &virtualmachinerelocate.ComputeParameter{DestinationID: &destinationID}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_VApp
// 资源池迁移
func Test_Relocate_VApp(t *testing.T) {
	VMID := "vm-1622"
	p := virtualmachinerelocate.RelocateParameter{}
	destinationID := "resgroup-v753"
	p.Compute = &virtualmachinerelocate.ComputeParameter{DestinationID: &destinationID}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_Host
// 主机迁移
func Test_Relocate_Host(t *testing.T) {
	// todo test
	VMID := "vm-1622"
	p := virtualmachinerelocate.RelocateParameter{}
	destinationID := "resgroup-v753"
	p.Compute = &virtualmachinerelocate.ComputeParameter{DestinationID: &destinationID}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_Host_Network
// 主机迁移和网络
func Test_Relocate_Host_Network(t *testing.T) {
	// todo test
	VMID := "vm-1622"
	p := virtualmachinerelocate.RelocateParameter{}
	destinationID := "resgroup-v753"
	p.Compute = &virtualmachinerelocate.ComputeParameter{DestinationID: &destinationID}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_Cluster
// 集群迁移
func Test_Relocate_Cluster(t *testing.T) {
	// todo test
	VMID := "vm-1622"
	p := virtualmachinerelocate.RelocateParameter{}
	destinationID := "resgroup-v753"
	p.Compute = &virtualmachinerelocate.ComputeParameter{DestinationID: &destinationID}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_Storage_uni
// 同一存储迁移
func Test_Relocate_Storage_uni(t *testing.T) {
	VMID := "vm-1087"
	p := virtualmachinerelocate.RelocateParameter{}
	datastoreID := "datastore-26"
	p.Storage = &virtualmachinerelocate.StorageParameter{
		DatastoreID: &datastoreID,
	}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}

// Test_Relocate_Storage_ind
// 独立存储迁移
func Test_Relocate_Storage_ind(t *testing.T) {
	VMID := "vm-1087"
	p := virtualmachinerelocate.RelocateParameter{}
	var disks []virtualmachinerelocate.DiskStorageParameter

	//disks = append(disks, virtualmachinerelocate.DiskStorageParameter{
	//	Key:         2000,
	//	DatastoreID: &datastoreID,
	//})
	datastoreID := "datastore-27"
	format := "thin"
	disks = append(disks, virtualmachinerelocate.DiskStorageParameter{
		Key:         2001,
		Format:      &format,
		DatastoreID: &datastoreID,
	})

	p.Storage = &virtualmachinerelocate.StorageParameter{
		Disks: disks,
	}
	_, err := virtualmachinerelocate.Relocate(vc.Api, VMID, p, 20)
	if err != nil {
		logging.L().Error(err)
	}
}
