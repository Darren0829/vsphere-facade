package security

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"vsphere_api/api/e"
	"vsphere_api/api/security/bearer"
	"vsphere_api/api/security/jwt"
	"vsphere_api/app/logging"
	"vsphere_api/config"
	"vsphere_api/vsphere"
)

const (
	CurrentAuth = "CURRENT_AUTH"
)

type Token interface {
	Generate(a vsphere.Auth) (string, error)
	Parse(t string) (*vsphere.Auth, error)
	Type() string
}

var tokenTool Token

func Setup() {
	tokenType := config.G.App.Token.Type
	tokenTool = getTokenTool(tokenType)
	setUpTokenTool()
	fmt.Println("当前使用token认证方式为:", tokenTool.Type())
}

func setUpTokenTool() {
	jwt.Setup()
}

// GetToken
// @Summary      获取令牌
// @Description  传入vcenter认证信息，获取令牌
// @Tags         认证
// @Accept       application/json
// @Produce      application/json
// @Param        object  body      vsphere.Auth  true  "认证信息"
// @Success      200     {string}  json          "{"code":"2000","message":"成功","data":{"token":""}}"
// @Failure      400     {string}  json          "{"code":"","message":"失败","data":{}"
// @Failure      500     {string}  json          "{"code":"","message":"失败","data":{}"
// @Router       /token [post]
func GetToken(c *gin.Context) {
	r := e.Gin{C: c}
	auth := vsphere.Auth{}
	err := c.ShouldBindBodyWith(&auth, binding.JSON)
	if err != nil {
		r.ResponseError(http.StatusBadRequest, e.BadRequest, nil)
		return
	}

	errors := e.ValidReqParam(&auth)
	if len(errors) > 0 {
		r.ResponseErrors(http.StatusBadRequest, errors, nil)
		return
	}

	vc := vsphere.Get(auth)
	if vc == nil {
		r.ResponseError(http.StatusBadRequest, e.ConnectFailed, nil)
		return
	}

	token, err := tokenTool.Generate(auth)
	if err != nil {
		r.ResponseError(http.StatusInternalServerError, e.SystemError, nil)
		return
	}

	r.ResponseOk(http.StatusOK, e.Success, map[string]string{
		"token": token,
	})
}

func Verify() gin.HandlerFunc {
	return func(c *gin.Context) {
		var code, message string
		code = e.Success
		token := c.GetHeader("token")
		logging.L().Debug("token: ", token)
		if token == "" {
			code = e.Unauthorized
			message = e.GetMessage(code)
		} else {
			auth, err := tokenTool.Parse(token)
			if err != nil {
				code = e.Unauthorized
				message = e.GetMessage(err.Error())
			} else {
				c.Set(CurrentAuth, *auth)
			}
		}

		if code != e.Success {
			c.JSON(http.StatusUnauthorized, e.Response{
				Code:    code,
				Message: message,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetCurrentAuth(c *gin.Context) vsphere.Auth {
	auth, _ := c.Get(CurrentAuth)
	return auth.(vsphere.Auth)
}

func getTokenTool(t string) Token {
	switch t {
	case jwt.Type:
		return jwt.Token{}
	default:
		return bearer.Token{}
	}
}
