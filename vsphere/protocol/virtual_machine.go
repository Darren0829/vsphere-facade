package protocol

import "time"

type VirtualMachineQuery struct {
	DatacenterID string
	FolderID     string
	ClusterID    string
	HostID       string
	IDs          []string
}

type VirtualMachineInfo struct {
	TemplateInfo

	IPAddress   string `json:"ip_address"`
	Hostname    string `json:"hostname"`
	PowerState  string `json:"power_state"`
	ToolsStatus string `json:"tools_status"`
}

type TemplateQuery struct {
	DatacenterID string
	FolderID     string
	IDs          []string
}

type TemplateInfo struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	UUID         string     `json:"uuid"`
	InstanceUUID string     `json:"instanceUUID"`
	Description  string     `json:"description"`
	CreateDate   *time.Time `json:"createDate,omitempty"`

	IPAddress         *string `json:"IPAddress,omitempty"`
	NumCPU            int32   `json:"numCPU"`
	NumCoresPerSocket int32   `json:"numCoresPerSocket"`
	MemoryMB          int32   `json:"memoryMB"`
	OSFamily          string  `json:"osFamily"`
	OSName            string  `json:"osName"`
	ToolsHasInstalled bool    `json:"toolsHasInstalled"`

	DatacenterID   string  `json:"datacenterId"`
	FolderID       *string `json:"folderId,omitempty"`
	ClusterID      *string `json:"clusterId,omitempty"`
	HostID         string  `json:"hostId"`
	ResourcePoolID *string `json:"resourcePoolId,omitempty"`

	SysDisk           DiskInfo               `json:"sysDisk"`
	DataDisks         []DiskInfo             `json:"dataDisks,omitempty"`
	NetworkInterfaces []NetworkInterfaceInfo `json:"networkInterfaces,omitempty"`
}

type DiskInfo struct {
	ID          string  `json:"id"`
	Key         int32   `json:"key"`
	Size        int32   `json:"size"`
	Mode        *string `json:"mode"`
	Format      *string `json:"format"`
	Sharing     *string `json:"sharing"`
	DatastoreID *string `json:"datastoreId"`
}

type NetworkInterfaceInfo struct {
	NetworkID   string   `json:"networkId"`
	ID          string   `json:"id"`
	Key         int32    `json:"key"`
	AdapterType string   `json:"adapterType"`
	MACAddress  string   `json:"macAddress,omitempty"`
	IPs         []IpInfo `json:"ipInfo,omitempty"`
}

type IpInfo struct {
	IpAddress string `json:"ipAddress"`
	State     string `json:"state"`
}
