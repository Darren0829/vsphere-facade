package protocol

type ClusterInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	DrsEnabled     bool   `json:"drsEnabled"`
	ResourcePoolId string `json:"resourcePoolId"`

	DatacenterID string `json:"datacenterId"`
}

type ClusterQuery struct {
	DatacenterID string
	IDs          []string
}

type OSFamilyInfo struct {
	ID              string
	Name            string
	Family          string
	SupportedMinMen int32
	SupportedMaxMen int32
	SupportedCPUs   int32
}

type DatacenterInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DatacenterQuery struct {
	IDs []string
}

type DatastoreInfo struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Accessible      bool     `json:"accessible"`
	Type            string   `json:"type"`
	Capacity        int64    `json:"capacity"`
	FreeSpace       int64    `json:"freeSpace"`
	Uncommitted     int64    `json:"uncommitted"`
	SupportDiskType []string `json:"supportDiskType,omitempty"`

	DatacenterID string `json:"datacenterId"`
}

type DatastoreQuery struct {
	DatacenterID string
	IDs          []string
}

type FolderInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`

	DatacenterID string `json:"datacenterID"`
}

type FolderQuery struct {
	DatacenterID string
	FolderID     string
	IDs          []string
}

type HostInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ResourcePoolId string   `json:"resourcePoolId"`
	ParentId       string   `json:"-"`
	Networks       []string `json:"networks"`
	Datastores     []string `json:"datastores"`

	DatacenterID string `json:"datacenterId"`
	ClusterID    string `json:"clusterId"`
}

type HostQuery struct {
	DatacenterID string
	ClusterID    string
	IDs          []string
}

type NetworkInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Accessible bool   `json:"accessible"`

	DatacenterID string `json:"datacenterId"`
}

type NetworkQuery struct {
	DatacenterID string
	IDs          []string
}

type ResourcePoolInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ParentType      string `json:"parentType"`
	ParentID        string `json:"parentId"`
	AvailableCpu    int64  `json:"availableCpu"`
	AvailableMemory int64  `json:"availableMemory"`

	DatacenterID string `json:"datacenterId"`
	ClusterID    string `json:"clusterId"`
	HostID       string `json:"hostId"`
}

type ResourcePoolQuery struct {
	DatacenterID string
	ClusterID    string
	HostID       string
	IDs          []string
}

type StoragePolicyQuery struct {
}

type StoragePolicyInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
