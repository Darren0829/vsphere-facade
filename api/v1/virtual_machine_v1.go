package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"vsphere-facade/api/e"
	"vsphere-facade/api/security"
	"vsphere-facade/app/logging"
	"vsphere-facade/vsphere"
	"vsphere-facade/vsphere/callback"
	"vsphere-facade/vsphere/protocol"
	"vsphere-facade/vsphere/workerpool"
	"vsphere-facade/vsphere/workerpool/taskreceiver"
)

type DeployReq struct {
	Parameter workerpool.DeployParameter `json:"config"  valid:"Required"`
	Timeout   *workerpool.TimeoutSetting `json:"timeout,omitempty"`

	CallBack protocol.CallbackReq `json:"callback,omitempty"`
}

type OperationReq struct {
	IDs []string `json:"ids" valid:"Required;MinSize(1)"`

	CallBack protocol.CallbackReq `json:"callback"`
}

type OperationCallBackRes struct {
	Success  []string          `json:"success,omitempty"`
	NotFound []string          `json:"not_found,omitempty"`
	Failed   []OperationFailed `json:"failed,omitempty"`
}

type DeploymentCallBackRes struct {
	IsSuccess bool        `json:"is_success"`
	Message   *string     `json:"message,omitempty"`
	Instance  interface{} `json:"instance,omitempty"`
}

type OperationFailed struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type OperationRes struct {
	RequestID string `json:"requestId"`
}

type DeployRes struct {
	RequestID string `json:"requestId"`
}

type VirtualMachineQuery struct {
	DatacenterID string   `json:"datacenterId"`
	FolderID     string   `json:"folderId"`
	ClusterID    string   `json:"clusterId"`
	HostID       string   `json:"hostId"`
	IDs          []string `json:"ids"`
}

type TemplateQuery struct {
	DatacenterID string   `form:"datacenterId"`
	FolderID     string   `form:"folderId"`
	IDs          []string `form:"ids"`
}

type RenameReq struct {
	ID      string `json:"id"`
	NewName string `json:"newName"`
}

type ReconfigureReq struct {
	ID string `json:"id"`
	workerpool.ReconfigureParameter

	CallBack protocol.CallbackReq `json:"callBack"`
}

type DiskReconfigureReq struct {
	ID string `json:"id"`
	workerpool.ReconfigureDiskParameter

	CallBack protocol.CallbackReq `json:"callBack"`
}

type NicReconfigureReq struct {
	ID string `json:"id"`
	workerpool.ReconfigureNicParameter

	CallBack protocol.CallbackReq `json:"callBack"`
}

type DescriptionReq struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type RelocateReq struct {
	ID string `json:"id"`
	workerpool.RelocateParameter

	CallBack protocol.CallbackReq `json:"callback,omitempty"`
}

// CreateVirtualMachine
// @Summary      创建虚拟机
// @Description  创建虚拟机
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.DeployReq  true  "创建参数"
// @Success      202  {object}  e.Response{data=[]v1.DeployRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines [post]
func CreateVirtualMachine(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := DeployReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, err.Error(), nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := DeployRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeDeployment, p)
	var vc = vsphere.Get(auth)
	vmDeployer := workerpool.NewVirtualMachineDeployer(vc.Api)
	vmDeployer.DeployID = res.RequestID
	vmDeployer.Parameter = p.Parameter
	vmDeployer.TimeoutSetting = p.Timeout
	errs := vmDeployer.Verify()
	if errs != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, errs)
		return
	}

	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeDeployment, func() {
		defer taskreceiver.Done(res.RequestID)
		var callBack = p.CallBack
		callBack.RequestID = vmDeployer.DeployID
		err := vmDeployer.Deploy()
		if err == nil {
			VMID := vmDeployer.NewMachineID()
			instanceInfo := vc.GetVirtualMachine(VMID)
			deploymentCallBack(callBack, DeploymentCallBackRes{
				IsSuccess: true,
				Instance:  instanceInfo,
			})
		} else {
			message := err.Error()
			deploymentCallBack(callBack, DeploymentCallBackRes{
				IsSuccess: false,
				Message:   &message,
			})
		}
	})
	if err != nil {
		logging.L().Error("创建部署任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// DeleteVirtualMachine
// @Summary      删除虚拟机
// @Description  删除虚拟机
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "删除虚拟机参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines [delete]
func DeleteVirtualMachine(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	var vc = vsphere.Get(auth)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		for _, ID := range p.IDs {
			machine := workerpool.GetVirtualMachineOperator(vc.Api, ID)
			if machine == nil {
				notFound = append(notFound, ID)
				continue
			}
			err := machine.Destroy()
			if err != nil {
				logging.L().Error("开机失败: ", err)
				failed = append(failed, OperationFailed{
					ID:    ID,
					Error: err.Error(),
				})
			} else {
				success = append(success, ID)
			}
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("删除任务创建失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// ModifyVirtualMachineConfigure
// @Summary      修改虚拟机配置
// @Description  修改虚拟机配置
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.ReconfigureReq  true  "修改虚拟机配置参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/reconfigure [post]
func ModifyVirtualMachineConfigure(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := ReconfigureReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	var vc = vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack

		err = machine.Reconfigure(p.ReconfigureParameter)
		if err != nil {
			logging.L().Error("修改配置失败", err)
			failed = append(failed, OperationFailed{
				ID:    p.ID,
				Error: err.Error(),
			})
		} else {
			success = append(success, p.ID)
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})
	if err != nil {
		logging.L().Error("创建配置修改任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// ReconfigureVirtualMachineNic
// @Summary      修改虚拟机网卡
// @Description  修改虚拟机网卡
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.NicReconfigureReq  true  "修改虚拟机网卡参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/reconfigure_nic [post]
func ReconfigureVirtualMachineNic(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := NicReconfigureReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	vc := vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		err := machine.ReconfigureNic(workerpool.ReconfigureNicParameter{
			Add:    p.Add,
			Edit:   p.Edit,
			Remove: p.Remove,
		})
		if err != nil {
			logging.L().Error("修改硬盘配置失败", err)
			failed = append(failed, OperationFailed{
				ID:    p.ID,
				Error: err.Error(),
			})
		} else {
			success = append(success, p.ID)
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("创建网卡修改任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// ReconfigureVirtualMachineDisk
// @Summary      修改虚拟机硬盘
// @Description  修改虚拟机硬盘
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.DiskReconfigureReq  true  "修改虚拟机硬盘参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/reconfigure_disk [post]
func ReconfigureVirtualMachineDisk(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := DiskReconfigureReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	vc := vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		err := machine.ReconfigureDisk(workerpool.ReconfigureDiskParameter{
			Add:    p.Add,
			Edit:   p.Edit,
			Remove: p.Remove,
		})
		if err != nil {
			logging.L().Error("修改硬盘配置失败", err)
			failed = append(failed, OperationFailed{
				ID:    p.ID,
				Error: err.Error(),
			})
		} else {
			success = append(success, p.ID)
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("创建硬盘修改任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// VirtualMachinePowerOn
// @Summary      开机
// @Description  开机
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "开机参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/power_on [post]
func VirtualMachinePowerOn(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	var vc = vsphere.Get(auth)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		for _, ID := range p.IDs {
			machine := workerpool.GetVirtualMachineOperator(vc.Api, ID)
			if machine == nil {
				notFound = append(notFound, ID)
				continue
			}
			err := machine.PowerOn()
			if err != nil {
				logging.L().Error("开机失败: ", err)
				failed = append(failed, OperationFailed{
					ID:    ID,
					Error: err.Error(),
				})
			} else {
				success = append(success, ID)
			}
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("创建开机任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// VirtualMachinePowerOff
// @Summary      关闭电源
// @Description  关闭电源
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "关闭电源参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/power_off [post]
func VirtualMachinePowerOff(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	var vc = vsphere.Get(auth)
	err = vc.AddTask(workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		for _, ID := range p.IDs {
			machine := workerpool.GetVirtualMachineOperator(vc.Api, ID)
			if machine == nil {
				notFound = append(notFound, ID)
				continue
			}
			err := machine.PowerOff()
			if err != nil {
				logging.L().Error("开机失败: ", err)
				failed = append(failed, OperationFailed{
					ID:    ID,
					Error: err.Error(),
				})
			} else {
				success = append(success, ID)
			}
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("关闭电源任务创建失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// VirtualMachineShutdown
// @Summary      关闭操作系统
// @Description  关闭操作系统
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "关闭操作系统参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/shutdown [post]
func VirtualMachineShutdown(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := OperationReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	var vc = vsphere.Get(auth)
	err = vc.AddTask(workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack
		for _, ID := range p.IDs {
			machine := workerpool.GetVirtualMachineOperator(vc.Api, ID)
			if machine == nil {
				notFound = append(notFound, ID)
				continue
			}
			err := machine.Shutdown()
			if err != nil {
				logging.L().Error("开机失败: ", err)
				failed = append(failed, OperationFailed{
					ID:    ID,
					Error: err.Error(),
				})
			} else {
				success = append(success, ID)
			}
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})

	if err != nil {
		logging.L().Error("关闭操作系统任务创建失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// VirtualMachineRelocate
// @Summary      虚拟机迁移
// @Description  虚拟机迁移
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.OperationReq  true  "虚拟机迁移参数"
// @Success      202  {object}  e.Response{data=[]v1.OperationRes}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/{id}/relocate [post]
func VirtualMachineRelocate(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := RelocateReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	var vc = vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}

	res := OperationRes{}
	res.RequestID = taskreceiver.Receive(workerpool.WorkerTypeOperation, p)
	err = workerpool.AddTask(vc.Api.ID, workerpool.WorkerTypeOperation, func() {
		defer taskreceiver.Done(res.RequestID)
		var success, notFound []string
		var failed []OperationFailed
		var callBack = p.CallBack

		err = machine.Relocate(p.RelocateParameter)
		if err != nil {
			logging.L().Error("迁移失败", err)
			failed = append(failed, OperationFailed{
				ID:    p.ID,
				Error: err.Error(),
			})
		} else {
			success = append(success, p.ID)
		}
		callBack.RequestID = res.RequestID
		operationCallBack(callBack, success, notFound, failed)
	})
	if err != nil {
		logging.L().Error("创建迁移任务失败: ", err)
		taskreceiver.Cancel(res.RequestID, "任务创建失败")
		r.ResponseOk(http.StatusInternalServerError, e.SystemError, nil)
	} else {
		r.ResponseOk(http.StatusAccepted, e.Accepted, res)
	}
}

// VirtualMachineRename
// @Summary      重命名
// @Description  重命名
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.RenameReq  true  "请求参数"
// @Success      200  {object}  e.Response
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/rename [post]
func VirtualMachineRename(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := RenameReq{}
	err := c.ShouldBind(&p)
	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, err.Error(), nil)
		return
	}

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	vc := vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		logging.L().Error("修改名称失败", err)
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}
	err = machine.Rename(p.NewName)
	if err != nil {
		logging.L().Error("修改名称失败", err)
		r.ResponseError(http.StatusBadRequest, err.Error(), nil)
		return
	}
	r.ResponseOk(http.StatusOK, e.Success, nil)
}

// VirtualMachineDescript
// @Summary      修改备注
// @Description  修改备注
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.DescriptionReq  true  "请求参数"
// @Success      200  {object}  e.Response
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines/description [post]
func VirtualMachineDescript(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	p := DescriptionReq{}
	err := c.ShouldBind(&p)

	errors := e.ValidReqParam(&p)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	if err != nil {
		logging.L().Error("解析请求参数出错: ", err)
		r.ResponseError(http.StatusBadRequest, err.Error(), nil)
		return
	}
	vc := vsphere.Get(auth)
	machine := workerpool.GetVirtualMachineOperator(vc.Api, p.ID)
	if machine == nil {
		logging.L().Error("修改备注失败", err)
		r.ResponseError(http.StatusBadRequest, e.VMNotFound, nil)
		return
	}
	err = machine.Descript(p.Description)
	if err != nil {
		logging.L().Error("修改备注失败", err)
		r.ResponseError(http.StatusBadRequest, err.Error(), nil)
		return
	}
	r.ResponseOk(http.StatusOK, e.Success, nil)
}

// QueryVirtualMachines
// @Summary      查询虚拟机
// @Description  查询虚拟机
// @Tags         虚拟机
// @Accept       json
// @Produce      json
// @Param        c    body      v1.VirtualMachineQuery  true  "查询参数"
// @Success      200  {object}  e.Response{data=[]protocol.VirtualMachineInfo}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/virtual_machines [get]
func QueryVirtualMachines(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := VirtualMachineQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&query)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	q := protocol.VirtualMachineQuery{
		DatacenterID: query.DatacenterID,
		FolderID:     query.FolderID,
		ClusterID:    query.ClusterID,
		HostID:       query.HostID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	templates := vc.QueryVirtualMachines(q)
	if templates == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, templates)
	}
}

// QueryTemplates
// @Summary      查询模板
// @Description  查询模板
// @Tags         模版
// @Accept       json
// @Produce      json
// @Param        c    body      v1.TemplateQuery  true  "查询参数"
// @Success      200  {object}  e.Response{data=[]protocol.TemplateInfo}
// @Failure      400  {string}  json  "{"code":"400x","message":"失败"}"
// @Failure      401  {string}  json  "{"code":"401x","message":"失败"}"
// @Failure      500  {string}  json  "{"code":"500x","message":"失败"}"
// @Security     ApiKeyAuth
// @Router       /v1/templates [get]
func QueryTemplates(c *gin.Context) {
	r := e.Gin{C: c}
	auth := security.GetCurrentAuth(c)

	query := TemplateQuery{}
	err := c.ShouldBind(&query)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&query)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	q := protocol.TemplateQuery{
		DatacenterID: query.DatacenterID,
		FolderID:     query.FolderID,
		IDs:          query.IDs,
	}
	var vc = vsphere.Get(auth)
	templates := vc.QueryTemplates(q)
	if templates == nil {
		r.ResponseOk(http.StatusOK, e.Success, e.EmptyArray())
	} else {
		r.ResponseOk(http.StatusOK, e.Success, templates)
	}
}

func operationCallBack(c protocol.CallbackReq, success, notFound []string, failed []OperationFailed) {
	cb := callback.NewCallbacker(c)
	cb.CallbackArr(c.RequestID, OperationCallBackRes{
		Success:  success,
		NotFound: notFound,
		Failed:   failed,
	})
}

func deploymentCallBack(c protocol.CallbackReq, res DeploymentCallBackRes) {
	cb := callback.NewCallbacker(c)
	cb.CallbackObj(c.RequestID, res)
}
