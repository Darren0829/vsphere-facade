package vsphere

import (
	"github.com/vmware/govmomi/vim25/mo"
	"vsphere_api/app/logging"
	"vsphere_api/app/utils"
	"vsphere_api/helper/folder"
	"vsphere_api/vsphere/protocol"
)

func (vc *VCenter) QueryFolders(q protocol.FolderQuery) []protocol.FolderInfo {
	folders := vc.queryFoldersFromCache(q)
	if folders != nil {
		logging.L().Debug("本次查询使用了缓存")
		return folders
	}

	if len(q.IDs) > 0 {
		return vc.getFoldersByIDs(q.IDs)
	} else if q.FolderID != "" {
		return vc.getFoldersByParentID(q.FolderID)
	} else if q.DatacenterID != "" {
		return vc.getVMFoldersByDatacenterID(q.DatacenterID)
	} else {
		return vc.getFolders()
	}
}

func (vc *VCenter) getFoldersByIDs(IDs []string) []protocol.FolderInfo {
	var FolderInfos []protocol.FolderInfo
	for _, ID := range IDs {
		moFolder := folder.GetMObject(vc.Api, ID)
		if moFolder == nil {
			continue
		}
		FolderInfo := vc.buildFolderInfo(*moFolder, "")
		FolderInfos = append(FolderInfos, FolderInfo)
	}
	return FolderInfos
}

func (vc *VCenter) getFoldersByParentID(FolderID string) []protocol.FolderInfo {
	moFolders := folder.GetByFolderID(vc.Api, FolderID)
	if moFolders == nil {
		return nil
	}
	var folderInfos []protocol.FolderInfo
	for _, moFolder := range *moFolders {
		FolderInfo := vc.buildFolderInfo(moFolder, "")
		folderInfos = append(folderInfos, FolderInfo)
	}
	return folderInfos
}

func (vc *VCenter) getVMFoldersByDatacenterID(datacenterID string) []protocol.FolderInfo {
	moFolders := folder.GetVMFoldersByDatacenterID(vc.Api, datacenterID)
	if moFolders == nil {
		return nil
	}
	var folderInfos []protocol.FolderInfo
	for _, moFolder := range *moFolders {
		FolderInfo := vc.buildFolderInfo(moFolder, datacenterID)
		folderInfos = append(folderInfos, FolderInfo)
	}
	return folderInfos
}

func (vc *VCenter) getFolders() []protocol.FolderInfo {
	dcs := vc.getDatacenters()
	if dcs == nil {
		return nil
	}
	var folderInfos []protocol.FolderInfo
	for _, dc := range dcs {
		datacenterID := dc.ID
		moFolders := folder.GetByDatacenterID(vc.Api, datacenterID)
		if moFolders == nil {
			continue
		}
		for _, moFolder := range *moFolders {
			FolderInfo := vc.buildFolderInfo(moFolder, datacenterID)
			folderInfos = append(folderInfos, FolderInfo)
		}
	}

	vc.Cache.CacheFolders(folderInfos)
	return folderInfos
}

func (vc *VCenter) buildFolderInfo(moFolder mo.Folder, datacenterID string) protocol.FolderInfo {
	var folderInfo protocol.FolderInfo
	folderInfo.DatacenterID = datacenterID
	folderInfo.ID = moFolder.Reference().Value
	folderInfo.Name = moFolder.Name
	folderInfo.ParentID = ""

	return folderInfo
}

func (vc *VCenter) queryFoldersFromCache(q protocol.FolderQuery) []protocol.FolderInfo {
	cache := vc.Cache.GetFolders()
	if cache == nil {
		return nil
	}

	if len(q.IDs) > 0 {
		var folders []protocol.FolderInfo
		for _, Folder := range cache {
			if utils.SliceContain(q.IDs, Folder.ID) {
				folders = append(folders, Folder)
			}
		}
		return folders
	} else if q.FolderID != "" {
		var Folders []protocol.FolderInfo
		for _, Folder := range cache {
			if Folder.ParentID == q.FolderID {
				Folders = append(Folders, Folder)
			}
		}
		return Folders
	} else if q.DatacenterID != "" {
		var Folders []protocol.FolderInfo
		for _, Folder := range cache {
			if Folder.DatacenterID == q.DatacenterID {
				Folders = append(Folders, Folder)
			}
		}
		return Folders
	} else {
		return cache
	}
}
