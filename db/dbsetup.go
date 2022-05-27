package db

import (
	"vsphere-facade/config"
	"vsphere-facade/db/badgerdb"
)

func Setup() {
	if config.G.Server.Db.Badger != nil {
		badgerdb.Setup()
	}
}
