// +build doc

package main

import (
	gs "github.com/swaggo/gin-swagger"
	"github.com/swaggo/gin-swagger/swaggerFiles"
)

func init() {
	swagHandler = gs.WrapHandler(swaggerFiles.Handler)
}
