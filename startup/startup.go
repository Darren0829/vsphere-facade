package startup

import (
	"encoding/json"
	"errors"
	"vsphere-facade/app/logging"
	"vsphere-facade/vsphere/callback"
	"vsphere-facade/vsphere/protocol"
	"vsphere-facade/vsphere/workerpool/taskreceiver"
)

func Run() {
	interruptTaskCallback()
}

// interruptTaskCallback
// 查询中断任务，并调用其回调，通知任务已中断
func interruptTaskCallback() {
	req := taskreceiver.GetReceivedReq()
	for requestID, p := range req {
		var pMap = make(map[string]interface{})
		err := json.Unmarshal([]byte(p), &pMap)
		if err != nil {
			logging.L().Errorf("%v", err)
			return
		}

		callbackParameter := pMap["callback"]
		logging.L().Debug(callbackParameter)
		b, err := json.Marshal(callbackParameter)
		if err != nil {
			logging.L().Errorf("%v", err)
			return
		}

		callbackReq := protocol.CallbackReq{}
		err = json.Unmarshal(b, &callbackReq)
		if err != nil {
			logging.L().Errorf("%v", err)
			return
		}

		callback.NewCallbacker(callbackReq).CallbackErr(requestID, nil, errors.New("系统中断，任务失去控制"))
		taskreceiver.Cancel(requestID, "系统中断，任务失去控制")
	}
}
