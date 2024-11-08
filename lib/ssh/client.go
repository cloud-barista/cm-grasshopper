package ssh

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	grasshopperCommon "github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/lib/config"

	"github.com/cloud-barista/cm-grasshopper/lib/rsautil"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"net"
	"strconv"
)

type ConnectionType int

const (
	ConnectionTypeSource ConnectionType = iota
	ConnectionTypeTarget
)

type Client struct {
	*goph.Client
	SSHTarget *model.SSHTarget
}

func AddKnownHost(host string, remote net.Addr, key ssh.PublicKey) error {
	hostFound, _ := goph.CheckKnownHost(host, remote, key, "")

	if hostFound {
		return nil
	}

	return goph.AddKnownHost(host, remote, key, "")
}

func decryptSecrets(connectionInfo *honeybee.ConnectionInfo) (*honeybee.ConnectionInfo, error) {
	encryptedUser, err := base64.StdEncoding.DecodeString(connectionInfo.User)
	if err != nil {
		errMsg := "error occurred while decrypting the base64 encoded encrypted user (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}

	decryptedUserBytes, err := rsautil.DecryptWithPrivateKey(encryptedUser, grasshopperCommon.HoneybeePrivateKey)
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

	decryptedPasswordBytes, err := rsautil.DecryptWithPrivateKey(encryptedPassword, grasshopperCommon.HoneybeePrivateKey)
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

	decryptedPrivateKeyBytes, err := rsautil.DecryptWithPrivateKey(encryptedPrivateKey, grasshopperCommon.HoneybeePrivateKey)
	if err != nil {
		errMsg := "error occurred while decrypting private key (" + err.Error() + ")"
		logger.Println(logger.ERROR, true, errMsg)

		return nil, errors.New(errMsg)
	}
	connectionInfo.PrivateKey = string(decryptedPrivateKeyBytes)

	return connectionInfo, nil
}

func NewSSHClient(connectionType ConnectionType, id string, nsID string, mciID string) (*Client, error) {
	var client *goph.Client
	var sshTarget *model.SSHTarget

	if connectionType == ConnectionTypeSource {
		if id == "" {
			return nil, errors.New("id is required")
		}

		data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
			":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
			"/honeybee/connection_info/"+id, "", "")
		if err != nil {
			return nil, err
		}

		var encryptedConnectionInfo honeybee.ConnectionInfo
		err = json.Unmarshal(data, &encryptedConnectionInfo)
		if err != nil {
			return nil, err
		}

		connectionInfo, err := decryptSecrets(&encryptedConnectionInfo)
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

		sshPort, _ := strconv.Atoi(connectionInfo.SSHPort)

		client, err = goph.NewConn(&goph.Config{
			User:     connectionInfo.User,
			Addr:     connectionInfo.IPAddress,
			Port:     uint(sshPort),
			Auth:     auth,
			Timeout:  goph.DefaultTimeout,
			Callback: AddKnownHost,
		})
		if err != nil {
			return nil, err
		}

		var useKeypair bool
		if connectionInfo.PrivateKey != "" && connectionInfo.PrivateKey != "-" {
			useKeypair = true
		}

		sshTarget = &model.SSHTarget{
			IP:         connectionInfo.IPAddress,
			Port:       uint(sshPort),
			UseKeypair: useKeypair,
			Username:   connectionInfo.User,
			Password:   connectionInfo.Password,
			PrivateKey: connectionInfo.PrivateKey,
		}
	} else if connectionType == ConnectionTypeTarget {
		if id == "" {
			return nil, errors.New("id is required")
		}
		if nsID == "" {
			return nil, errors.New("nsId is required")
		}
		if mciID == "" {
			return nil, errors.New("mciId is required")
		}

		data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
			":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
			"/tumblebug/ns/"+nsID+"/mci/"+mciID+"/vm/"+id,
			config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
		if err != nil {
			return nil, err
		}

		var vmInfo model.TBVMInfo
		err = json.Unmarshal(data, &vmInfo)
		if err != nil {
			return nil, err
		}

		sshPort, err := strconv.Atoi(vmInfo.SSHPort)
		if err != nil {
			return nil, errors.New("invalid ssh port")
		}

		data, err = common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
			":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
			"/tumblebug/ns/"+nsID+"/resources/sshKey/"+vmInfo.SSHKeyID,
			config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
		if err != nil {
			return nil, err
		}

		var sshKeyInfo model.TBSSHKeyInfo
		err = json.Unmarshal(data, &sshKeyInfo)
		if err != nil {
			return nil, err
		}

		if sshKeyInfo.PrivateKey == "" {
			return nil, errors.New("failed to get private key")
		}

		var auth goph.Auth
		auth, err = goph.RawKey(sshKeyInfo.PrivateKey, "")
		if err != nil {
			return nil, err
		}

		client, err = goph.NewConn(&goph.Config{
			User:     vmInfo.VMUserName,
			Addr:     vmInfo.PublicIP,
			Port:     uint(sshPort),
			Auth:     auth,
			Timeout:  goph.DefaultTimeout,
			Callback: AddKnownHost,
		})
		if err != nil {
			return nil, err
		}

		sshTarget = &model.SSHTarget{
			IP:         vmInfo.PublicIP,
			Port:       uint(sshPort),
			UseKeypair: true,
			Username:   vmInfo.VMUserName,
			Password:   "",
			PrivateKey: sshKeyInfo.PrivateKey,
		}
	} else {
		return nil, errors.New("invalid connection type")
	}

	return &Client{
		client,
		sshTarget,
	}, nil
}
