package ssh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	grasshopperCommon "github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/rsautil"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"net"
)

type Client struct {
	*goph.Client
	ConnectionInfo honeybee.ConnectionInfo
}

func AddKnownHost(host string, remote net.Addr, key ssh.PublicKey) error {
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")

	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	if hostFound && err != nil {
		return err
	}

	if hostFound {
		return nil
	}

	return goph.AddKnownHost(host, remote, key, "")
}

func decryptPasswordAndPrivateKey(connectionInfo *honeybee.ConnectionInfo) (*honeybee.ConnectionInfo, error) {
	encryptedPassword, err := base64.StdEncoding.DecodeString(connectionInfo.Password)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted password (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	decryptedPasswordBytes := rsautil.DecryptWithPrivateKey(encryptedPassword, grasshopperCommon.HoneybeePrivateKey)
	connectionInfo.Password = string(decryptedPasswordBytes)

	encryptedPrivateKey, err := base64.StdEncoding.DecodeString(connectionInfo.PrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted private key (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)
		return nil, errors.New(errMsg)
	}

	decryptedPrivateKeyBytes := rsautil.DecryptWithPrivateKey(encryptedPrivateKey, grasshopperCommon.HoneybeePrivateKey)
	connectionInfo.PrivateKey = string(decryptedPrivateKeyBytes)

	return connectionInfo, nil
}

func NewSSHClient(connectionInfoID string) (*Client, error) {
	data, err := common.GetHTTPRequest("http://" + config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress +
		":" + config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort +
		"/honeybee/connection_info/" + connectionInfoID)
	if err != nil {
		return nil, err
	}

	var encryptedConnectionInfo honeybee.ConnectionInfo
	err = json.Unmarshal(data, &encryptedConnectionInfo)
	if err != nil {
		return nil, err
	}

	connectionInfo, err := decryptPasswordAndPrivateKey(&encryptedConnectionInfo)
	if err != nil {
		return nil, err
	}

	var auth goph.Auth
	if connectionInfo.PrivateKey != "" && connectionInfo.PrivateKey != "-" {
		auth, err = goph.RawKey(connectionInfo.PrivateKey, "")
		if err != nil {
			return nil, err
		}
	} else if connectionInfo.Password != "" {
		auth = goph.Password(connectionInfo.Password)
	} else {
		return nil, errors.New("failed to determine auth method")
	}

	client, err := goph.NewConn(&goph.Config{
		User:     connectionInfo.User,
		Addr:     connectionInfo.IPAddress,
		Port:     uint(connectionInfo.SSHPort),
		Auth:     auth,
		Timeout:  goph.DefaultTimeout,
		Callback: AddKnownHost,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		client,
		*connectionInfo,
	}, nil
}
