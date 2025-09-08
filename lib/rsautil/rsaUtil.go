package rsautil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/jollaman999/utils/logger"
)

// BytesToPrivateKey bytes to private key
func BytesToPrivateKey(priv []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(priv)
	b := block.Bytes
	var err error
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// DecryptWithPrivateKey decrypts data with private key
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	hash := sha512.New()

	chunkSize := priv.Size()

	var plaintextData bytes.Buffer
	offset := 0

	for offset < len(ciphertext) {
		end := offset + chunkSize
		if end > len(ciphertext) {
			errMsg := "invalid ciphertext length"
			logger.Println(logger.ERROR, true, errMsg)
			return nil, errors.New(errMsg)
		}

		chunk := ciphertext[offset:end]
		plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, priv, chunk, nil)
		if err != nil {
			logger.Println(logger.ERROR, true, err.Error())
			return nil, err
		}

		_, err = plaintextData.Write(plaintext)
		if err != nil {
			logger.Println(logger.ERROR, true, err.Error())
			return nil, err
		}

		offset = end
	}

	return plaintextData.Bytes(), nil
}
