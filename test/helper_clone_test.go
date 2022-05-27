package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"vsphere_api/helper/virtualmachine"
	"vsphere_api/helper/virtualmachine/virtualmachineclone"
)

func Test_clone_1(t *testing.T) {
	cloneFromMOID := "vm-641"
	name := "Test_clone_1"
	datacenterID := "datacenter-2"
	folderID := "group-v557"
	hostID := "host-517"

	p := virtualmachineclone.CloneParameter{}
	p.ID = cloneFromMOID
	p.Name = name
	p.Location = virtualmachineclone.LocationParameter{
		DatacenterID: datacenterID,
		FolderID:     &folderID,
		HostId:       &hostID,
	}

	oVM, err := virtualmachineclone.Clone(vc.Api, p, 20)
	assert.NoError(t, err)

	props := virtualmachine.FindAllProps(oVM)
	assert.NotNil(t, props)
}
