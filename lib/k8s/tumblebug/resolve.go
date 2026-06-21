// Package tumblebug resolves kubeconfigs for cb-tumblebug-managed Kubernetes clusters
// into a form usable by cm-grasshopper's in-container client-go (no cloud CLI available).
package tumblebug

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	restcommon "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/jollaman999/utils/logger"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type kubeconfigResponse struct {
	Kubeconfig string `json:"kubeconfig"`
}

func baseURL() (string, error) {
	tb := config.CMGrasshopperConfig.CMGrasshopper.Tumblebug
	if strings.TrimSpace(tb.ServerAddress) == "" || strings.TrimSpace(tb.ServerPort) == "" {
		return "", errors.New("cb-tumblebug server address/port is not configured")
	}
	return "http://" + tb.ServerAddress + ":" + tb.ServerPort, nil
}

// ResolveKubeconfig fetches the kubeconfig of a cb-tumblebug-managed k8s cluster and
// returns a kubeconfig that client-go can use inside the cm-grasshopper container.
//
// cb-tumblebug (through cb-spider) returns one of two kinds of kubeconfig:
//   - self-contained, with embedded client cert/key/token (e.g. Azure AKS); used as-is.
//   - exec-plugin based, shelling out to a cloud CLI such as aws-iam-authenticator or
//     gke-gcloud-auth-plugin (e.g. AWS EKS, GCP GKE, NCP NKS). Those CLIs and their cloud
//     credentials do not exist in this container, so the exec stanza is rewritten into a
//     "token broker" form that pulls a short-lived bearer token from cb-tumblebug's /token
//     endpoint. client-go re-runs the broker command when the token expires (or on 401),
//     so long-running velero backup/restore keep working.
func ResolveKubeconfig(nsID, k8sClusterID string) (string, error) {
	nsID = strings.TrimSpace(nsID)
	k8sClusterID = strings.TrimSpace(k8sClusterID)
	if nsID == "" || k8sClusterID == "" {
		return "", errors.New("cb-tumblebug namespaceId and k8sClusterId are required")
	}

	base, err := baseURL()
	if err != nil {
		return "", err
	}

	tb := config.CMGrasshopperConfig.CMGrasshopper.Tumblebug
	kubeconfigURL := fmt.Sprintf("%s/tumblebug/ns/%s/k8sCluster/%s/kubeconfig", base, nsID, k8sClusterID)
	data, err := restcommon.GetHTTPRequest(kubeconfigURL, tb.Username, tb.Password)
	if err != nil {
		return "", fmt.Errorf("failed to fetch kubeconfig from cb-tumblebug: %w", err)
	}

	var resp kubeconfigResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse cb-tumblebug kubeconfig response: %w (body: %s)", err, truncate(string(data), 200))
	}
	if strings.TrimSpace(resp.Kubeconfig) == "" {
		return "", fmt.Errorf("cb-tumblebug returned empty kubeconfig (body: %s)", truncate(string(data), 200))
	}

	apiCfg, err := clientcmd.Load([]byte(resp.Kubeconfig))
	if err != nil {
		return "", fmt.Errorf("failed to load cb-tumblebug kubeconfig: %w", err)
	}

	if !rewriteExecToBroker(apiCfg, base, nsID, k8sClusterID, tb.Username, tb.Password) {
		// Self-contained kubeconfig (e.g. Azure AKS); nothing to rewrite.
		return resp.Kubeconfig, nil
	}

	out, err := clientcmd.Write(*apiCfg)
	if err != nil {
		return "", fmt.Errorf("failed to serialize rewritten kubeconfig: %w", err)
	}

	logger.Println(logger.INFO, false,
		"tumblebug.ResolveKubeconfig: rewrote exec-plugin kubeconfig to cb-tumblebug token broker for ns='"+
			nsID+"' k8sCluster='"+k8sClusterID+"'")

	return string(out), nil
}

// rewriteExecToBroker replaces every exec-plugin based user credential with a token-broker
// exec stanza that fetches a bearer token from cb-tumblebug. It returns true if at least one
// user was rewritten (i.e. the kubeconfig was exec-plugin based).
func rewriteExecToBroker(cfg *clientcmdapi.Config, base, nsID, k8sClusterID, user, pass string) bool {
	tokenURL := fmt.Sprintf("%s/tumblebug/ns/%s/k8sCluster/%s/token", base, nsID, k8sClusterID)

	// cb-tumblebug wraps the ExecCredential under an "execCredential" key, but client-go
	// expects the bare ExecCredential object on stdout, so unwrap it with jq.
	shellCmd := fmt.Sprintf(`curl -fsS -u %s %s | jq -ce .execCredential`,
		shellQuote(user+":"+pass), shellQuote(tokenURL))

	broker := &clientcmdapi.ExecConfig{
		APIVersion:      "client.authentication.k8s.io/v1",
		Command:         "sh",
		Args:            []string{"-c", shellCmd},
		InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
	}

	rewritten := false
	for name, authInfo := range cfg.AuthInfos {
		if authInfo == nil || authInfo.Exec == nil {
			continue
		}
		newAuth := *authInfo
		newAuth.Exec = broker
		newAuth.AuthProvider = nil
		cfg.AuthInfos[name] = &newAuth
		rewritten = true
	}
	return rewritten
}

// shellQuote wraps a string in single quotes for safe use inside `sh -c`.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
