package db

import (
	"vsphere_api/config"
	"vsphere_api/db/badgerdb"
)

func Setup() {
	if config.G.Server.Db.Badger != nil {
		badgerdb.Setup()
	}
}
