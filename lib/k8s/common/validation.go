package common

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/k8s/tumblebug"
	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
)

func DecodeKubeconfig(value string) (string, error) {
	if value == "" {
		return "", errors.New("kubeconfig is required")
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil && strings.TrimSpace(string(decoded)) != "" {
		return string(decoded), nil
	}

	return value, nil
}

func DefaultNamespace(cluster *commonmodel.ClusterAccess, fallback string) string {
	if cluster == nil || strings.TrimSpace(cluster.Namespace) == "" {
		return fallback
	}
	return cluster.Namespace
}

func ValidateClusterAccess(cluster *commonmodel.ClusterAccess) error {
	if cluster == nil {
		return errors.New("cluster access is required")
	}
	if strings.TrimSpace(cluster.Kubeconfig) == "" && cluster.Tumblebug == nil {
		return errors.New("either kubeconfig or tumblebug cluster reference is required")
	}
	if cluster.Tumblebug != nil {
		if strings.TrimSpace(cluster.Tumblebug.NamespaceID) == "" ||
			strings.TrimSpace(cluster.Tumblebug.K8sClusterID) == "" {
			return errors.New("tumblebug namespaceId and k8sClusterId are required")
		}
	}
	return nil
}

// ResolveKubeconfig returns the effective kubeconfig for a cluster: the directly supplied
// kubeconfig when present, otherwise one resolved from cb-tumblebug (with exec-plugin CSPs
// rewritten to a self-contained token-broker form).
func ResolveKubeconfig(cluster *commonmodel.ClusterAccess) (string, error) {
	if err := ValidateClusterAccess(cluster); err != nil {
		return "", err
	}
	if strings.TrimSpace(cluster.Kubeconfig) != "" {
		return DecodeKubeconfig(cluster.Kubeconfig)
	}
	return tumblebug.ResolveKubeconfig(cluster.Tumblebug.NamespaceID, cluster.Tumblebug.K8sClusterID)
}

func ValidateS3Access(s3 *commonmodel.S3Access) error {
	if s3 == nil {
		return errors.New("s3 access is required")
	}
	if strings.TrimSpace(s3.Endpoint) == "" {
		return errors.New("s3 endpoint is required")
	}
	if strings.TrimSpace(s3.AccessKey) == "" {
		return errors.New("s3 accessKey is required")
	}
	if strings.TrimSpace(s3.SecretKey) == "" {
		return errors.New("s3 secretKey is required")
	}

	_, _, err := NormalizeS3Endpoint(s3)
	if err != nil {
		return err
	}

	return nil
}

func NormalizeS3Endpoint(s3 *commonmodel.S3Access) (string, bool, error) {
	if s3 == nil {
		return "", false, errors.New("s3 access is required")
	}

	rawEndpoint := strings.TrimSpace(s3.Endpoint)
	if rawEndpoint == "" {
		return "", false, errors.New("s3 endpoint is required")
	}

	if strings.Contains(rawEndpoint, "://") {
		return "", false, errors.New("s3 endpoint must not include scheme; use host[:port] only and set useSSL separately")
	}

	if strings.ContainsAny(rawEndpoint, "/?#") {
		return "", false, errors.New("s3 endpoint must not include path, query string, or fragment")
	}

	normalized := strings.TrimSuffix(rawEndpoint, "/")
	if normalized == "" {
		return "", false, errors.New("s3 endpoint is required")
	}

	return normalized, s3.UseSSL, nil
}

func BuildS3URL(s3 *commonmodel.S3Access) (string, error) {
	endpoint, useSSL, err := NormalizeS3Endpoint(s3)
	if err != nil {
		return "", err
	}

	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, endpoint), nil
}

func DefaultS3Bucket(s3 *commonmodel.S3Access, fallback string) string {
	if s3 == nil || strings.TrimSpace(s3.Bucket) == "" {
		return fallback
	}

	return strings.TrimSpace(s3.Bucket)
}
