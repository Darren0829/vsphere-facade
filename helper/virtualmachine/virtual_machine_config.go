package virtualmachine

const (
	Disk     = "Disk"
	Ethernet = "Ethernet"
)

var DeviceStartKey = map[string]int32{
	Disk:     2000,
	Ethernet: 4000,
}
