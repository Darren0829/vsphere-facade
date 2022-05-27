package datastore

import (
	"vsphere_api/helper"
	"vsphere_api/helper/disk"
)

const (
	TypeVMFS  = "VMFS"
	TypeNFS   = "NFS"
	TypeNFS41 = "NFS41"
	TypeVSAN  = "VSAN"
	TypeVVOL  = "VVOL"
)

var SupportedFormatOptions = map[string][]string{
	TypeVMFS:  {disk.FormatThin, disk.FormatFlat, disk.FormatThink, "sameAsSource"},
	TypeNFS:   {disk.FormatThin},
	TypeNFS41: {disk.FormatThin},
	TypeVSAN:  {"asDefinedInProfile"},
	TypeVVOL:  {disk.FormatThin, disk.FormatNativeThink},
}

func GetSupportedFormats(api *helper.API, datastoreID string) {

}
