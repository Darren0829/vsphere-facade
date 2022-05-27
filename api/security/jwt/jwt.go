package jwt

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"time"
	"vsphere-facade/api/e"
	"vsphere-facade/app/utils"
	"vsphere-facade/config"
	"vsphere-facade/vsphere"
)

var jwtSecret []byte

const Type = "jwt"

type Claims struct {
	Auth vsphere.Auth
	jwt.StandardClaims
}

type Token struct {
}

func Setup() {
	jwtSecret = []byte(config.G.App.Token.Secret)
}

func (t Token) Generate(a vsphere.Auth) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(3 * time.Hour)

	claims := Claims{
		a,
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "vsphere-facade",
		},
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(jwtSecret)
	return token, err
}

func (t Token) Parse(token string) (*vsphere.Auth, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		switch err.(*jwt.ValidationError).Errors {
		case jwt.ValidationErrorExpired:
			return nil, fmt.Errorf(e.TokenExpired)
		default:
			return nil, fmt.Errorf(e.Unauthorized)
		}
	}

	if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
		auth := vsphere.Auth{
			Address:  utils.AesDecrypt(claims.Auth.Address),
			Username: utils.AesDecrypt(claims.Auth.Username),
			Password: utils.AesDecrypt(claims.Auth.Password),
		}
		return &auth, nil
	} else {
		return nil, fmt.Errorf(e.TokenInvalid)
	}
}

func (t Token) Type() string {
	return Type
}
