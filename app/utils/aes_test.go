package utils

import (
	"fmt"
	"testing"
)

func TestAesEncrypt(t *testing.T) {
	en := AesEncrypt("administrator@vsphere.local")

	fmt.Println(en)
}

func TestAesDecrypt(t *testing.T) {
	de := AesDecrypt("3PxvnuL02IaQOz63pGxIhYbblUbZrHmxKOCAmA7zYjY=")
	fmt.Println(de)
}
