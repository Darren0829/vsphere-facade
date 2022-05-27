package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"vsphere-facade/app/logging"
)

var (
	key = []byte("480055b0a0c4d10c")
)

func AesEncrypt(plaintext string) string {
	block, err := aes.NewCipher(key)
	if err != nil {
		logging.L().Error("", errors.New("加密失败，Key无效"))
		return ""
	}

	var b = []byte(plaintext)

	blockSize := block.BlockSize()
	b = PKCS5Padding(b, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key)

	ciphertext := make([]byte, len(b))
	blockMode.CryptBlocks(ciphertext, b)

	return base64.StdEncoding.EncodeToString(ciphertext)
}

func AesDecrypt(ciphertext string) string {
	block, err := aes.NewCipher(key)
	if err != nil {
		logging.L().Error("", errors.New("解密失败，Key无效"))
		return ""
	}

	b, _ := base64.StdEncoding.DecodeString(ciphertext)
	blockSize := block.BlockSize()

	if len(b) < blockSize {
		logging.L().Error("", errors.New("解密失败，解密源无效"))
		return ""
	}

	if len(b)%blockSize != 0 {
		logging.L().Error("", errors.New("解密失败，解密源无效"))
		return ""
	}

	blockModel := cipher.NewCBCDecrypter(block, key)

	plaintext := make([]byte, len(b))
	blockModel.CryptBlocks(plaintext, b)
	plaintext = PKCS5UnPadding(plaintext)

	return string(plaintext)
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}
