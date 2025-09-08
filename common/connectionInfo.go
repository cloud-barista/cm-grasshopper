package common

import (
	"encoding/base64"
	"errors"

	"github.com/cloud-barista/cm-grasshopper/lib/rsautil"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

func ConnectionInfoDecryptSecrets(connectionInfo *honeybee.ConnectionInfo) (*honeybee.ConnectionInfo, error) {
	encryptedUser, err := base64.StdEncoding.DecodeString(connectionInfo.User)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted user (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}

	decryptedUserBytes, err := rsautil.DecryptWithPrivateKey(encryptedUser, HoneybeePrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting user (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}
	connectionInfo.User = string(decryptedUserBytes)

	encryptedPassword, err := base64.StdEncoding.DecodeString(connectionInfo.Password)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted password (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}

	decryptedPasswordBytes, err := rsautil.DecryptWithPrivateKey(encryptedPassword, HoneybeePrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting password (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}
	connectionInfo.Password = string(decryptedPasswordBytes)

	encryptedPrivateKey, err := base64.StdEncoding.DecodeString(connectionInfo.PrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted private key (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}

	decryptedPrivateKeyBytes, err := rsautil.DecryptWithPrivateKey(encryptedPrivateKey, HoneybeePrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting private key (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}
	connectionInfo.PrivateKey = string(decryptedPrivateKeyBytes)

	return connectionInfo, nil
}
