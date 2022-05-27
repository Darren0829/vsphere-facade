package vsphere

import (
	"context"
	"fmt"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/pbm"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"time"
	"vsphere-facade/app/logging"
	"vsphere-facade/helper"
)

func GetPbmClient(api *helper.API, ctx context.Context) *pbm.Client {
	pc, err := pbm.NewClient(ctx, api.Client.Client)
	if err != nil {
		logging.L().Error("创建pbm客户端时发生错误", err)
		return nil
	}
	return pc
}

func WaitForTask(oTask *object.Task, timeout int32) error {
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*time.Duration(timeout))
		defer cancel()
		return oTask.Wait(ctx)
	}
	return nil
}

func FindParent(api *helper.API, id, oType string) *types.ManagedObjectReference {
	ref := types.ManagedObjectReference{
		Type:  oType,
		Value: id,
	}

	var m mo.ManagedEntity
	ctx, cancel := context.WithTimeout(context.Background(), helper.APITimeout)
	defer cancel()
	err := property.DefaultCollector(api.Client.Client).RetrieveOne(ctx, ref, []string{"parent"}, &m)
	if err != nil {
		logging.L().Error(fmt.Sprintf("获取管理对象[%s:%s]时,发生错误", oType, id), err)
		return nil
	}
	return m.Parent
}

func FindParentByType(api *helper.API, id, oType, parentType string) *types.ManagedObjectReference {
	parent := FindParent(api, id, oType)
	if parent == nil {
		return nil
	}

	if parent.Type == parentType {
		return parent
	}

	return FindParentByType(api, parent.Value, parent.Type, parentType)
}

func FindParentPathByType(api *helper.API, id, oType, parentType string) []types.ManagedObjectReference {
	var parents []types.ManagedObjectReference
	parent := FindParent(api, id, oType)
	if parent == nil {
		return nil
	}

	parents = append(parents, parent.Reference())
	if parent.Type == parentType {
		return parents
	}

	findParents := FindParentPathByType(api, parent.Value, parent.Type, parentType)
	parents = append(parents, findParents...)
	return parents
}
