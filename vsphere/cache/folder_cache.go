package cache

import (
	"vsphere_api/helper/folder"
	"vsphere_api/vsphere/protocol"
)

func (c VCCache) CacheFolders(v []protocol.FolderInfo) {
	c.Set(folder.Type, v)
}

func (c VCCache) GetFolder(ID string) *protocol.FolderInfo {
	Folders := c.GetFolders()
	if Folders != nil {
		for _, c := range Folders {
			if c.ID == ID {
				return &c
			}
		}
	}
	return nil
}

func (c VCCache) GetFolders() []protocol.FolderInfo {
	v, b := c.Get(folder.Type)
	if !b {
		return nil
	}
	return v.([]protocol.FolderInfo)
}
