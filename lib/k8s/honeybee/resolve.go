// Package honeybee resolves kubeconfigs stored on cm-honeybee connections (on-prem k8s
// sources registered with ResourceType k8s).
package honeybee

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	comm "github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	restcommon "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	honeybeemodel "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

func baseURL() (string, error) {
	hb := config.CMGrasshopperConfig.CMGrasshopper.Honeybee
	if strings.TrimSpace(hb.ServerAddress) == "" || strings.TrimSpace(hb.ServerPort) == "" {
		return "", errors.New("cm-honeybee server address/port is not configured")
	}
	return "http://" + hb.ServerAddress + ":" + hb.ServerPort, nil
}

// ResolveKubeconfig fetches a cm-honeybee connection and returns its decrypted kubeconfig.
//
// cm-honeybee validates the kubeconfig as YAML on registration and returns it RSA encrypted
// and base64 encoded, so the decrypted value is a raw kubeconfig YAML string, the same form
// tumblebug.ResolveKubeconfig returns.
func ResolveKubeconfig(connID string) (string, error) {
	connID = strings.TrimSpace(connID)
	if connID == "" {
		return "", errors.New("honeybee connectionId is required")
	}

	base, err := baseURL()
	if err != nil {
		return "", err
	}

	data, err := restcommon.GetHTTPRequest(base+"/honeybee/connection_info/"+connID, "", "")
	if err != nil {
		return "", fmt.Errorf("failed to fetch connection info from cm-honeybee: %w", err)
	}

	var encryptedConnectionInfo honeybeemodel.ConnectionInfo
	if err := json.Unmarshal(data, &encryptedConnectionInfo); err != nil {
		return "", fmt.Errorf("failed to parse cm-honeybee connection info response: %w", err)
	}

	connectionInfo, err := comm.ConnectionInfoDecryptSecrets(&encryptedConnectionInfo)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(connectionInfo.Kubeconfig) == "" {
		return "", fmt.Errorf("honeybee connection '%s' has no kubeconfig", connID)
	}

	return connectionInfo.Kubeconfig, nil
}
