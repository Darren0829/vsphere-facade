package workerpool

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"vsphere_api/app/cache"
	"vsphere_api/app/logging"
	"vsphere_api/config"
)

type WorkerType string

const (
	WorkerTypeOperation  = WorkerType("operation")
	WorkerTypeDeployment = WorkerType("deployment")
)

var receiveTaskPool *ants.Pool

func init() {
	receiveTaskPool, _ = ants.NewPool(10000,
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(0))
}

func Get(VCID string, t WorkerType) *ants.Pool {
	k := poolKey(VCID, t)
	p, exist := cache.INST.Get(k)
	if exist {
		return p.(*ants.Pool)
	}
	return newPool(VCID, t)
}

func AddTask(VCID string, t WorkerType, task func()) error {
	return receiveTaskPool.Submit(func() {
		err := Get(VCID, t).Submit(task)
		if err != nil {
			logging.L().Error("添加任务失败： ", err)
		}
	})
}

func newPool(VCID string, t WorkerType) *ants.Pool {
	var m sync.Mutex
	m.Lock()
	defer m.Unlock()

	k := poolKey(VCID, t)
	p, exist := cache.INST.Get(k)
	if exist {
		return p.(*ants.Pool)
	}

	switch t {
	case WorkerTypeDeployment:
		pool, err := ants.NewPool(config.G.Vsphere.RoutineCount.Deployment,
			ants.WithNonblocking(false),
			ants.WithMaxBlockingTasks(0))
		if err != nil {
			logging.L().Panic("创建工作池失败", err)
			return nil
		}
		k := poolKey(VCID, t)
		cache.INST.Set(k, pool, -1)
		return pool
	case WorkerTypeOperation:
		pool, err := ants.NewPool(config.G.Vsphere.RoutineCount.Operation,
			ants.WithNonblocking(false),
			ants.WithMaxBlockingTasks(0))
		if err != nil {
			logging.L().Panic("创建工作池失败", err)
			return nil
		}
		k := poolKey(VCID, t)
		cache.INST.Set(k, pool, -1)
		return pool
	}
	logging.L().Panic("创建工作池失败", fmt.Errorf("不识别的工作池类型[%s]", t))
	return nil
}

func poolKey(VCID string, t WorkerType) string {
	return fmt.Sprintf("%s::%s", VCID, t)
}
