package badgerdb

import (
	"github.com/dgraph-io/badger/v3"
	"log"
	"vsphere_api/app/logging"
	"vsphere_api/app/utils"
	"vsphere_api/config"
)

var db *badger.DB

func Setup() {
	dataPath := config.G.Server.Db.Badger.Path
	if dataPath != "" {
		var err error
		db, err = badger.Open(badger.DefaultOptions(dataPath))
		if err != nil {
			logging.L().Panic("badger DB初始化失败", err)
		}
	}
}

func Set(k, v string) {
	if !isAvailable() {
		return
	}

	wb := db.NewWriteBatch()
	defer wb.Cancel()
	err := wb.SetEntry(badger.NewEntry([]byte(k), []byte(v)).WithMeta(0))
	if err != nil {
		log.Println("Failed to write data to cache.", "key", k, "value", v, "err", err)

	}

	err = wb.Flush()
	if err != nil {
		log.Println("Failed to flush data to cache.", "key", k, "value", v, "err", err)
	}
}

func Get(k string) string {
	if !isAvailable() {
		return ""
	}

	var val []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(k))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		logging.L().Errorf("读取键[%s]的值错误：%v", k, err)
	}
	return string(val)
}

func Has(k string) bool {
	return false
}

func Del(k string) error {
	if !isAvailable() {
		return nil
	}

	wb := db.NewWriteBatch()
	defer wb.Cancel()
	err := wb.Delete([]byte(k))
	if err != nil {
		logging.L().Errorf("删除键[%s]出错: %v", k, err)
		return err
	}
	err = wb.Flush()
	if err != nil {
		logging.L().Errorf("删除键[%s]Flush出错: %v", k, err)
		return err
	}
	return err
}

func GetAll() map[string]string {
	var all = make(map[string]string)
	if !isAvailable() {
		return all
	}

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				logging.L().Errorf("提取键[%s]值出错: %v", k, err)
				continue
			}
			all[string(k)] = string(v)
		}
		return nil
	})

	if err != nil {
		logging.L().Error(err)
	}
	return all
}

func TableInfo() {
	for _, info := range db.Tables() {
		logging.L().Error(utils.ToJson(info))
	}
}

func isAvailable() bool {
	if db == nil {
		logging.L().Debug("badger DB未启用")
		return false
	}
	return true
}
