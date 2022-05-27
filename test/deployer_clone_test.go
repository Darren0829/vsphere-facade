package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"vsphere_api/helper/virtualmachine"
	"vsphere_api/helper/virtualmachine/virtualmachineclone"
	"vsphere_api/helper/virtualmachine/virtualmachinecustomize"
	"vsphere_api/vsphere/workerpool"
)

// Test_clone_default_value
//
func Test_clone_default_value(t *testing.T) {
	templateID := "vm-641"
	name := "Test_clone_default_value"
	datacenterID := "datacenter-2"
	datastoreID := "datastore-519"
	hostID := "host-517"
	format := "thin"
	gateway := "192.168.25.1"
	subnetMask := int32(24)
	ipv4 := "192.168.25.205"

	d := workerpool.NewVirtualMachineDeployer(vc.Api)
	d.Parameter = workerpool.DeployParameter{
		Name: name,
		Template: workerpool.Template{
			ID: templateID,
			SysDisk: &workerpool.SysDisk{
				DatastoreId: &datastoreID,
				Format:      &format,
			},
		},
		Location: virtualmachineclone.LocationParameter{
			DatacenterID: datacenterID,
			HostId:       &hostID,
		},
		NetworkInterfaces: []*workerpool.NetworkInterface{&workerpool.NetworkInterface{
			NetworkID:  "network-18",
			Gateway:    []string{gateway},
			SubnetMask: &subnetMask,
			IPv4: &virtualmachinecustomize.NicIPv4Setting{
				Static:    true,
				IPAddress: &ipv4,
			},
		}},
		DataDisks: []*workerpool.DataDisk{&workerpool.DataDisk{
			DatastoreId: datastoreID,
			Format:      format,
			Size:        20,
		}},
	}
	err := d.Deploy()

	assert.NoError(t, err)
	if err != nil {
		return
	}

	moVM := virtualmachine.GetMObject(vc.Api, d.NewMachineID())
	assert.Equal(t, moVM.Name, name)
}
