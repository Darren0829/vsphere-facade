package e

import (
	"encoding/json"
	"github.com/astaxie/beego/validation"
	"vsphere-facade/app/logging"
)

type ReqParamError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

func ValidReqParam(obj interface{}) []ReqParamError {
	if logging.IsDebug() {
		b, _ := json.Marshal(obj)
		logging.L().Debugf("请求参数: %s", string(b))
	}
	var errors []ReqParamError
	valid := validation.Validation{}
	ok, _ := valid.Valid(obj)
	if !ok {
		for _, err := range valid.Errors {
			logging.L().Error(err.Key, err.Message)
			errors = append(errors, ReqParamError{
				Key:     err.Key,
				Message: err.Message,
			})
		}
	}
	return errors
}
