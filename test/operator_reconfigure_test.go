package test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/types"
	"testing"
	"vsphere-facade/helper/virtualmachine"
	"vsphere-facade/vsphere/workerpool"
)

// Test_reconfigure_cold_cpu
// 条件: 开机状态下 关闭CPU热插拔
// 测试: 修改CPU核心数
func Test_reconfigure_cold_cpu(t *testing.T) {
	VMID := "vm-1624"
	numCPU := int32(4)

	o := workerpool.GetVirtualMachineOperator(vc.Api, VMID)

	p := workerpool.ReconfigureParameter{}
	p.NumCPU = numCPU

	err := o.Reconfigure(p)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	oVM := virtualmachine.GetObject(vc.Api, VMID)
	moVM := virtualmachine.FindProps(oVM)

	assert.Equal(t, numCPU, moVM.Config.Hardware.NumCPU)
	assert.Equal(t, types.VirtualMachinePowerStatePoweredOn, moVM.Runtime.PowerState)
}

// Test_reconfigure_hot_cpu
// 条件: 开机状态下 开启CPU热插拔
// 测试: 修改CPU核心数
func Test_reconfigure_hot_cpu(t *testing.T) {
	VMID := "vm-1624"
	numCPU := int32(4)

	o := workerpool.GetVirtualMachineOperator(vc.Api, VMID)

	p := workerpool.ReconfigureParameter{}
	p.NumCPU = numCPU

	err := o.Reconfigure(p)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	oVM := virtualmachine.GetObject(vc.Api, VMID)
	moVM := virtualmachine.FindProps(oVM)

	assert.Equal(t, numCPU, moVM.Config.Hardware.NumCPU)
	assert.Equal(t, types.VirtualMachinePowerStatePoweredOn, moVM.Runtime.PowerState)
}

// Test_reconfigure_hot_cpu_cold_mem
// 条件: 开机状态下 开启CPU热插拔 关闭内存热插拔
// 测试: 修改CPU核心数
func Test_reconfigure_hot_cpu_cold_mem(t *testing.T) {
	VMID := "vm-1624"
	numCPU := int32(4)
	memoryMB := int32(1024 * 5)

	o := workerpool.GetVirtualMachineOperator(vc.Api, VMID)

	p := workerpool.ReconfigureParameter{}
	p.NumCPU = numCPU
	p.MemoryMB = memoryMB

	err := o.Reconfigure(p)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	oVM := virtualmachine.GetObject(vc.Api, VMID)
	moVM := virtualmachine.FindProps(oVM)

	assert.Equal(t, numCPU, moVM.Config.Hardware.NumCPU)
	assert.Equal(t, memoryMB, moVM.Config.Hardware.MemoryMB)
	assert.Equal(t, types.VirtualMachinePowerStatePoweredOn, moVM.Runtime.PowerState)
}

// Test_reconfigure_hot_cpu_hot_mem
// 条件: 开机状态下 开启CPU热减少
// 测试: 减少CPU核心数，减少内存
func Test_reconfigure_powered_on_hot_remove_cpu_remove_memory(t *testing.T) {
	VMID := "vm-1624"
	numCPU := int32(2)
	memoryMB := int32(1024 * 4)

	o := workerpool.GetVirtualMachineOperator(vc.Api, VMID)

	p := workerpool.ReconfigureParameter{}
	p.NumCPU = numCPU
	p.MemoryMB = memoryMB

	err := o.Reconfigure(p)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	oVM := virtualmachine.GetObject(vc.Api, VMID)
	moVM := virtualmachine.FindProps(oVM)

	assert.Equal(t, numCPU, moVM.Config.Hardware.NumCPU)
	assert.Equal(t, memoryMB, moVM.Config.Hardware.MemoryMB)
	assert.Equal(t, types.VirtualMachinePowerStatePoweredOn, moVM.Runtime.PowerState)
}
