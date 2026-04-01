package common

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	restmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
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

func DefaultNamespace(cluster *restmodel.ClusterAccess, fallback string) string {
	if cluster == nil || strings.TrimSpace(cluster.Namespace) == "" {
		return fallback
	}
	return cluster.Namespace
}

func ValidateClusterAccess(cluster *restmodel.ClusterAccess) error {
	if cluster == nil {
		return errors.New("cluster access is required")
	}
	if strings.TrimSpace(cluster.Kubeconfig) == "" {
		return errors.New("kubeconfig is required")
	}
	return nil
}

func ValidateMinIOAccess(minio *restmodel.MinIOAccess) error {
	if minio == nil {
		return errors.New("minio access is required")
	}
	if strings.TrimSpace(minio.Endpoint) == "" {
		return errors.New("minio endpoint is required")
	}
	if strings.TrimSpace(minio.AccessKey) == "" {
		return errors.New("minio accessKey is required")
	}
	if strings.TrimSpace(minio.SecretKey) == "" {
		return errors.New("minio secretKey is required")
	}

	_, _, err := NormalizeMinIOEndpoint(minio)
	if err != nil {
		return err
	}

	return nil
}

func NormalizeMinIOEndpoint(minio *restmodel.MinIOAccess) (string, bool, error) {
	if minio == nil {
		return "", false, errors.New("minio access is required")
	}

	rawEndpoint := strings.TrimSpace(minio.Endpoint)
	if rawEndpoint == "" {
		return "", false, errors.New("minio endpoint is required")
	}

	if strings.Contains(rawEndpoint, "://") {
		return "", false, errors.New("minio endpoint must not include scheme; use host[:port] only and set useSSL separately")
	}

	if strings.ContainsAny(rawEndpoint, "/?#") {
		return "", false, errors.New("minio endpoint must not include path, query string, or fragment")
	}

	normalized := strings.TrimSuffix(rawEndpoint, "/")
	if normalized == "" {
		return "", false, errors.New("minio endpoint is required")
	}

	return normalized, minio.UseSSL, nil
}

func BuildMinIOS3URL(minio *restmodel.MinIOAccess) (string, error) {
	endpoint, useSSL, err := NormalizeMinIOEndpoint(minio)
	if err != nil {
		return "", err
	}

	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, endpoint), nil
}

func DefaultMinIOBucket(minio *restmodel.MinIOAccess, fallback string) string {
	if minio == nil || strings.TrimSpace(minio.Bucket) == "" {
		return fallback
	}

	return strings.TrimSpace(minio.Bucket)
}
