package ssh

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"syscall"

	"net"
	"strconv"
	"time"

	comm "github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"

	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

type ConnectionType int

const (
	ConnectionTypeSource ConnectionType = iota
	ConnectionTypeTarget
)

type Client struct {
	*goph.Client
	SSHTarget      *model.SSHTarget
	connectionType ConnectionType
	id             string
	nsID           string
	infraID        string
	keepAliveStop  chan struct{}
	keepAliveOnce  sync.Once
}

// IsConnBroken reports whether err indicates the underlying SSH transport is
// dead (as opposed to a command-level failure). Long migrations open thousands
// of sessions against the same connection; the target sshd, a network blip or an
// idle-timeout can tear the transport down mid-run. When that happens every later
// session/command fails until the connection is rebuilt, so these errors must
// trigger a reconnect rather than a plain retry.
//
// Network-layer failures are matched against real error values via errors.Is/As
// (io.EOF, net.ErrClosed, the syscall errnos, the net.Error timeout interface),
// which is robust to wrapping. golang.org/x/crypto/ssh reports many channel- and
// session-level transport failures as plain unwrapped strings, so a lower-cased
// substring check is kept as a fallback for those.
func IsConnBroken(err error) bool {
	if err == nil {
		return false
	}

	// Typed / sentinel errors (wrapping-safe).
	if errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Fallback: the SSH library surfaces these as unwrapped strings.
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"eof",
		"connection reset",
		"broken pipe",
		"use of closed network connection",
		"connection lost",
		"connection closed",
		"session creation failed",
		"failed to create session",
		"unexpected packet",
		"disconnect",
		"handshake failed",
		"connection refused",
		"i/o timeout",
		"no route to host",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// reconnect tears down the current transport and rebuilds it, updating the
// embedded goph client in place so callers keep using the same *Client.
func (c *Client) reconnect() error {
	if c.Client != nil {
		_ = c.Client.Close()
	}

	newClient, err := NewSSHClient(c.connectionType, c.id, c.nsID, c.infraID)
	if err != nil {
		return err
	}

	c.Client = newClient.Client
	return nil
}

func (c *Client) NewSessionWithRetry() (*ssh.Session, error) {
	var session *ssh.Session
	var err error

	// Try to create session with existing connection first
	for retry := 0; retry < 5; retry++ {
		session, err = c.NewSession()
		if err == nil {
			return session, nil
		}

		// If the transport looks dead, rebuild the connection before retrying.
		if IsConnBroken(err) {
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				time.Sleep(time.Second * 2)
				continue
			}
			continue
		}

		// For other errors, just retry after delay
		time.Sleep(time.Second * 2)
	}

	return nil, err
}

// RunWithRetry runs cmd on the host and returns its combined output. Unlike the
// embedded goph Run, it rebuilds the connection and retries when the transport
// dies mid-run, so a single reset does not cascade into every remaining step.
func (c *Client) RunWithRetry(cmd string) ([]byte, error) {
	var out []byte
	var err error

	for retry := 0; retry < 5; retry++ {
		out, err = c.Run(cmd)
		if err == nil {
			return out, nil
		}

		if IsConnBroken(err) {
			if reconnectErr := c.reconnect(); reconnectErr != nil {
				time.Sleep(time.Second * 2)
			}
			continue
		}

		return out, err
	}

	return out, err
}

func (c *Client) startKeepAlive() {
	c.keepAliveOnce.Do(func() {
		c.keepAliveStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if c.Client != nil {
						_, _, _ = c.SendRequest("keepalive@"+comm.ShortModuleName, true, nil)
					}
				case <-c.keepAliveStop:
					return
				}
			}
		}()
	})
}

func (c *Client) Close() error {
	if c.keepAliveStop != nil {
		close(c.keepAliveStop)
	}
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}

func AddKnownHost(host string, remote net.Addr, key ssh.PublicKey) error {
	hostFound, _ := goph.CheckKnownHost(host, remote, key, "")

	if hostFound {
		return nil
	}

	return goph.AddKnownHost(host, remote, key, "")
}

func NewSSHClient(connectionType ConnectionType, id string, nsID string, infraID string) (*Client, error) {
	var client *goph.Client
	var sshTarget *model.SSHTarget

	switch connectionType {
	case ConnectionTypeSource:
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

		connectionInfo, err := comm.ConnectionInfoDecryptSecrets(&encryptedConnectionInfo)
		if err != nil {
			return nil, err
		}

		var auth goph.Auth
		if connectionInfo.PrivateKey != "" && connectionInfo.PrivateKey != "-" {
			privateKey := strings.ReplaceAll(connectionInfo.PrivateKey, "\\n", "\n")
			auth, err = goph.RawKey(privateKey, "")
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
	case ConnectionTypeTarget:
		if id == "" {
			return nil, errors.New("id is required")
		}
		if nsID == "" {
			return nil, errors.New("nsId is required")
		}
		if infraID == "" {
			return nil, errors.New("infraId is required")
		}

		data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
			":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
			"/tumblebug/ns/"+nsID+"/infra/"+infraID+"/node/"+id,
			config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
		if err != nil {
			return nil, err
		}

		var nodeInfo model.TBNodeInfo
		err = json.Unmarshal(data, &nodeInfo)
		if err != nil {
			return nil, err
		}

		sshPort := nodeInfo.SSHPort

		data, err = common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
			":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
			"/tumblebug/ns/"+nsID+"/resources/sshKey/"+nodeInfo.SSHKeyID,
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
			User:     nodeInfo.NodeUserName,
			Addr:     nodeInfo.PublicIP,
			Port:     uint(sshPort),
			Auth:     auth,
			Timeout:  goph.DefaultTimeout,
			Callback: AddKnownHost,
		})
		if err != nil {
			return nil, err
		}

		sshTarget = &model.SSHTarget{
			IP:         nodeInfo.PublicIP,
			Port:       uint(sshPort),
			UseKeypair: true,
			Username:   nodeInfo.NodeUserName,
			Password:   "",
			PrivateKey: sshKeyInfo.PrivateKey,
		}
	default:
		return nil, errors.New("invalid connection type")
	}

	sshClient := &Client{
		Client:         client,
		SSHTarget:      sshTarget,
		connectionType: connectionType,
		id:             id,
		nsID:           nsID,
		infraID:        infraID,
	}

	// Start SSH KeepAlive to prevent connection timeout
	sshClient.startKeepAlive()

	return sshClient, nil
}
