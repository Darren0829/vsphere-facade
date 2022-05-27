package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"vsphere_api/config"
)

func Test_config_storagePolicies_read(t *testing.T) {
	storagePolicies := config.G.Vsphere.Default.Deployment.StoragePolicies
	assert.Equal(t, "p0011", storagePolicies["vc0004"]["vmfs"])
	assert.Equal(t, "p0012", storagePolicies["vc0001"]["vsan"])
}
