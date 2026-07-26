package crypto

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func PKCS7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - len(data)%blockSize
	padding := make([]byte, padLen)
	for i := range padding {
		padding[i] = byte(padLen)
	}
	return append(data, padding...)
}

func PKCS7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen < 1 || padLen > len(data) || padLen > 16 {
		return nil, errors.New("invalid padding")
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, errors.New("invalid padding")
		}
	}
	return data[:len(data)-padLen], nil
}

func AESECBEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := PKCS7Pad(plaintext, block.BlockSize())
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += block.BlockSize() {
		block.Encrypt(dst[i:i+block.BlockSize()], padded[i:i+block.BlockSize()])
	}
	return dst, nil
}

func AESECBDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, errors.New("ciphertext not multiple of block size")
	}
	dst := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += block.BlockSize() {
		block.Decrypt(dst[i:i+block.BlockSize()], ciphertext[i:i+block.BlockSize()])
	}
	return PKCS7Unpad(dst)
}

func RSAEncryptPKCS1v15(message []byte, pemPublicKey string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemPublicKey))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsa.EncryptPKCS1v15(rand.Reader, rsaPub, message)
}

func GenerateDeviceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	hex := make([]byte, 32)
	const hextable = "0123456789ABCDEF"
	for i, v := range b {
		hex[i*2] = hextable[v>>4]
		hex[i*2+1] = hextable[v&0x0f]
	}
	return string(hex)
}
