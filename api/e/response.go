package e

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

type Gin struct {
	C *gin.Context
}

type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type EmptyData struct {
}

func (g *Gin) ResponseOk(httpCode int, errCode string, data interface{}) {
	g.C.JSON(httpCode, Response{
		Code:    errCode,
		Message: GetMessage(errCode),
		Data:    data,
	})
	return
}

func (g *Gin) ResponseError(httpCode int, errCode string, data interface{}) {
	g.C.JSON(httpCode, Response{
		Code:    errCode,
		Message: GetMessage(errCode),
		Data:    data,
	})
	return
}

func (g *Gin) ResponseErrors(httpCode int, errors []ReqParamError, data interface{}) {
	var errMsg string
	for i, err := range errors {
		if i == 0 {
			errMsg = err.Message
		} else {
			errMsg = fmt.Sprintf("%s;%s", errMsg, err.Message)
		}
	}
	g.C.JSON(httpCode, Response{
		Code:    BadRequest,
		Message: errMsg,
		Data:    data,
	})
	return
}

func EmptyArray() []EmptyData {
	return []EmptyData{}
}

func EmptyObject() EmptyData {
	return EmptyData{}
}
