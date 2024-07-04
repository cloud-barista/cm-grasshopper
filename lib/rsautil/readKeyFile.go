package rsautil

import (
	"crypto/rsa"
	"os"
)

func ReadPrivateKey(privateKeyFilePath string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(privateKeyFilePath)
	if err != nil {
		return nil, err
	}

	privKey, err := BytesToPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}
