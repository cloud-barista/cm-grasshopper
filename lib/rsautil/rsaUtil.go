package rsautil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"github.com/labstack/gommon/log"
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
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) []byte {
	hash := sha512.New()

	chunkSize := priv.Size()

	if len(ciphertext) == 0 {
		log.Error("ciphertext is empty")
		return nil
	}

	var plaintextData bytes.Buffer
	offset := 0

	for offset < len(ciphertext) {
		end := offset + chunkSize
		if end > len(ciphertext) {
			log.Error("invalid ciphertext length")
			return nil
		}

		chunk := ciphertext[offset:end]
		plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, priv, chunk, nil)
		if err != nil {
			log.Error(err)
			return nil
		}

		_, err = plaintextData.Write(plaintext)
		if err != nil {
			log.Error(err)
			return nil
		}

		offset = end
	}

	return plaintextData.Bytes()
}
