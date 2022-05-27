package cache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var INST *cache.Cache

func Setup() {
	INST = cache.New(time.Hour*2, time.Minute)
}
