package taskreceiver

import (
	"fmt"
	"github.com/google/uuid"
	"vsphere-facade/app/logging"
	"vsphere-facade/app/utils"
	"vsphere-facade/db/badgerdb"
	"vsphere-facade/vsphere/workerpool"
)

func Receive(reqType workerpool.WorkerType, req interface{}) string {
	ID := generateReqID(reqType)
	reqJson := utils.ToJson(req)
	logging.L().Debugf("收到一个请求，ID: %s, Req: %s", ID, reqJson)
	badgerdb.Set(ID, reqJson)
	return ID
}

func Done(ID string) {
	logging.L().Debug(fmt.Sprintf("请求[%s]完成", ID))
	_ = badgerdb.Del(ID)
}

func Cancel(ID string, reason string) {
	logging.L().Debug(fmt.Sprintf("请求[%s]取消，原因: %s", ID, reason))
	_ = badgerdb.Del(ID)
}

func GetReceivedReq() map[string]string {
	return badgerdb.GetAll()
}

func generateReqID(reqType workerpool.WorkerType) string {
	return fmt.Sprintf("%s:%s", string(reqType), uuid.NewString())
}
