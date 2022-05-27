package bearer

import (
	"encoding/json"
	"fmt"
	"vsphere_api/api/e"
	"vsphere_api/app/utils"
	"vsphere_api/vsphere"
)

const Type = "bearer"

type Token struct {
}

func (t Token) Generate(a vsphere.Auth) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return utils.AesEncrypt(string(b)), nil
}

func (t Token) Parse(token string) (*vsphere.Auth, error) {
	var a vsphere.Auth
	deToken := utils.AesDecrypt(token)
	err := json.Unmarshal([]byte(deToken), &a)
	if err != nil {
		return nil, fmt.Errorf(e.TokenInvalid)
	}
	return &a, nil
}

func (t Token) Type() string {
	return Type
}
