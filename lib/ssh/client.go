package ssh

import (
	"encoding/json"
	"errors"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
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

func NewSSHClient(connectionInfoUUID string) (*Client, error) {
	data, err := common.GetHTTPRequest("http://" + config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress +
		":" + config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort +
		"/connection_info/" + connectionInfoUUID)
	if err != nil {
		return nil, err
	}

	var connectionInfo honeybee.ConnectionInfo
	err = json.Unmarshal(data, &connectionInfo)
	if err != nil {
		return nil, err
	}

	var auth goph.Auth
	if connectionInfo.PrivateKey != "" {
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
		connectionInfo,
	}, nil
}
