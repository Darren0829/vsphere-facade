package helper

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi"
	"net/url"
	"strconv"
	"strings"
	"time"
	"vsphere_api/app/logging"
	"vsphere_api/config"
)

var apiCache = make(map[string]*API)
var APITimeout time.Duration

type API struct {
	ID      string
	Type    string
	version *APIVersion
	Client  *govmomi.Client
}

type APIVersion struct {
	Major int
	Minor int
	Patch int
	V     int
}

func Setup() {
	APITimeout = time.Duration(config.G.Vsphere.Timeout.Api) * time.Minute
}

func Init(address, username, password string) *API {
	logging.L().Debug("初始化VC连接")
	n := time.Now()
	// 先从缓存里拿
	k := cacheKey(address, username, password)
	cacheApi := getFromCache(k)
	if cacheApi != nil {
		logging.L().Debug("获取到缓存连接")
		return cacheApi
	}

	// 新建
	newApi := API{}
	u, err := url.Parse(address + "/sdk")
	if err != nil {
		logging.L().Panic("连接测试时发生错误", err)
	}
	u.User = url.UserPassword(username, password)
	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		logging.L().Panic(err)
	}
	newApi.Client = client
	newApi.parseVer()
	newApi.checkVer()
	cache(k, &newApi)
	logging.L().Debug("初始化VC连接完成，耗时: ", time.Since(n))
	return &newApi
}

func (a *API) Newer(major, minor, patch int) bool {
	if a.version == nil {
		a.parseVer()
	}
	v := version(major, minor, patch)
	ret := v <= a.version.V
	if !ret {
		logging.L().Debug(fmt.Sprintf("当前访问VCenter版本为[%d %d.%d]，小于[%d %d.%d]",
			a.version.Major, a.version.Minor, a.version.Patch,
			major, minor, patch))
	}
	return ret
}

func (a *API) parseVer() {
	about := a.Client.ServiceContent.About
	ver := about.ApiVersion
	s := strings.Split(ver, ".")
	major, _ := strconv.Atoi(s[0])
	minor, _ := strconv.Atoi(s[1])
	patch, _ := strconv.Atoi(s[2])
	a.version = &APIVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		V:     version(major, minor, patch),
	}
	a.Type = about.ApiType
	a.ID = about.InstanceUuid
}

func (a *API) checkVer() {
	if !a.Newer(5, 5, 0) {
		ver := a.version
		current := fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
		logging.L().Panicf("支持的最小版本为[5.5.0], 当前版本为[%s]", current)
	}
}

func version(major, minor, patch int) int {
	return major*1000 + minor*100 + patch
}

func getFromCache(k string) *API {
	a := apiCache[k]
	if a != nil && a.Client != nil && a.Client.Valid() {
		ctx, cancel := context.WithTimeout(context.Background(), APITimeout)
		defer cancel()
		ok, _ := a.Client.SessionManager.SessionIsActive(ctx)
		if ok {
			return a
		}
	}
	return nil
}

func cache(k string, a *API) {
	apiCache[k] = a
}

func cacheKey(address, username, password string) string {
	return fmt.Sprint(address, ":", username, ":", password)
}
