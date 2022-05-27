package cache

import (
	"fmt"
	"strings"
	"vsphere-facade/app/cache"
	"vsphere-facade/app/logging"
	"vsphere-facade/config"
)

const RefreshTicker = "RefreshTicker"

var ignorePrefixes []string

type VCCache struct {
	VCID string
}

func Setup() {
	ignore := config.G.Vsphere.Cache.Ignore
	if ignore != nil {
		for _, i := range ignore {
			if len(i.Items) > 0 {
				for _, item := range i.Items {
					ignorePrefixes = append(ignorePrefixes, key(i.VCID, item))
				}
			} else {
				ignorePrefixes = append(ignorePrefixes, i.VCID)
			}
		}
	}
}

func (c VCCache) Get(k string) (interface{}, bool) {
	if config.G.Vsphere.Cache.Enable {
		ck := key(c.VCID, k)
		return cache.INST.Get(ck)
	}
	return nil, false
}

func (c VCCache) Set(k string, v interface{}) {
	if config.G.Vsphere.Cache.Enable {
		ck := key(c.VCID, k)
		if !ignore(ck) {
			_, b := cache.INST.Get(ck)
			if !b {
				cache.INST.Set(ck, v, -1)
			}
		}
	}
}

func (c VCCache) CleanAll() {
	logging.L().Debug(fmt.Sprintf("清除VCenter[%s]下所有缓存数据", c.VCID))
	for k := range cache.INST.Items() {
		if strings.HasPrefix(k, c.VCID) && !strings.HasSuffix(k, RefreshTicker) {
			cache.INST.Delete(k)
		}
	}
}

func (c VCCache) Clean(keys ...string) {
	if keys != nil {
		logging.L().Debug(fmt.Sprintf("清除VCenter[%s]缓存数据: [%s]", c.VCID, keys))
		for _, k := range keys {
			cache.INST.Delete(key(c.VCID, k))
		}
	}
}

func key(VCID, t string) string {
	return fmt.Sprintf("%s::%s", VCID, t)
}

func ignore(k string) bool {
	for _, prefix := range ignorePrefixes {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}
