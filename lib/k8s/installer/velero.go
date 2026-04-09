package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	k8sclient "github.com/cloud-barista/cm-grasshopper/lib/k8s/client"
	k8scommon "github.com/cloud-barista/cm-grasshopper/lib/k8s/common"
	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
	veleromodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/velero"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultVeleroNamespace = "velero"
	DefaultVeleroBucket    = "velero"
	veleroChartURL         = "https://github.com/vmware-tanzu/helm-charts/releases/download/velero-12.0.0/velero-12.0.0.tgz"
	veleroChartVersion     = "12.0.0"
	veleroImageTag         = "v1.18.0"
)

type InstallResult struct {
	Status           string        `json:"status"`
	Message          string        `json:"message"`
	Namespace        string        `json:"namespace"`
	Bucket           string        `json:"bucket"`
	VolumeBackupMode string        `json:"volumeBackupMode"`
	InstallationTime time.Duration `json:"installation_time"`
}

type VeleroInstaller struct{}

func NewVeleroInstaller() *VeleroInstaller {
	return &VeleroInstaller{}
}

func (v *VeleroInstaller) Install(ctx context.Context, cluster *commonmodel.ClusterAccess, minioAccess *commonmodel.MinIOAccess, force bool, volumeBackupMode string) (*InstallResult, error) {
	start := time.Now()
	namespace := k8scommon.DefaultNamespace(cluster, DefaultVeleroNamespace)
	bucketName := k8scommon.DefaultMinIOBucket(minioAccess, DefaultVeleroBucket)
	if volumeBackupMode == "" {
		volumeBackupMode = veleromodel.VolumeBackupModeFilesystem
	}

	clientset, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	if err := v.ensureNamespace(ctx, clientset, namespace); err != nil {
		return nil, err
	}

	minioClient, err := k8sclient.NewMinIOClient(minioAccess)
	if err != nil {
		return nil, err
	}

	if err := k8sclient.EnsureMinIOBucket(ctx, minioClient, bucketName); err != nil {
		return nil, fmt.Errorf("failed to ensure minio bucket %q: %w", bucketName, err)
	}

	if err := v.ensureSecret(ctx, clientset, namespace, minioAccess, force); err != nil {
		return nil, err
	}

	actionConfig, err := k8sclient.NewHelmActionConfig(&commonmodel.ClusterAccess{
		Kubeconfig: cluster.Kubeconfig,
		Namespace:  namespace,
	})
	if err != nil {
		return nil, err
	}

	if err := v.installOrUpgradeChart(actionConfig, namespace, minioAccess, force, volumeBackupMode); err != nil {
		return nil, err
	}

	if err := v.waitForDeploymentReady(ctx, clientset, namespace); err != nil {
		return nil, err
	}

	if err := v.ensureBackupStorageLocation(ctx, controllerClient, namespace, minioAccess, force); err != nil {
		return nil, err
	}

	return &InstallResult{
		Status:           "completed",
		Message:          "Velero installed successfully",
		Namespace:        namespace,
		Bucket:           bucketName,
		VolumeBackupMode: volumeBackupMode,
		InstallationTime: time.Since(start),
	}, nil
}

func (v *VeleroInstaller) ensureNamespace(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	_, err = clientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}, metav1.CreateOptions{})
	return err
}

func (v *VeleroInstaller) ensureSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace string, minioAccess *commonmodel.MinIOAccess, force bool) error {
	secretName := "cloud-credentials"

	if force {
		_ = clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	}

	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"cloud": fmt.Sprintf(`[default]
aws_access_key_id=%s
aws_secret_access_key=%s
region=minio
`, minioAccess.AccessKey, minioAccess.SecretKey),
		},
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

func (v *VeleroInstaller) installOrUpgradeChart(actionConfig *action.Configuration, namespace string, minioAccess *commonmodel.MinIOAccess, force bool, volumeBackupMode string) error {
	chartRef, err := downloadChart(veleroChartURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(filepath.Dir(chartRef))
	}()

	loadedChart, err := loader.Load(chartRef)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	values := buildVeleroValues(minioAccess, volumeBackupMode)

	get := action.NewGet(actionConfig)
	_, err = get.Run("velero")
	if err == nil {
		upgrade := action.NewUpgrade(actionConfig)
		upgrade.Namespace = namespace
		upgrade.Install = false
		upgrade.Wait = true
		upgrade.Timeout = 10 * time.Minute
		_, err = upgrade.Run("velero", loadedChart, values)
		return err
	}

	if force || strings.Contains(err.Error(), "release: not found") {
		install := action.NewInstall(actionConfig)
		install.ReleaseName = "velero"
		install.Namespace = namespace
		install.CreateNamespace = true
		install.Version = veleroChartVersion
		install.Wait = true
		install.Timeout = 10 * time.Minute
		_, err = install.Run(loadedChart, values)
		return err
	}

	return err
}

func buildVeleroValues(minioAccess *commonmodel.MinIOAccess, volumeBackupMode string) map[string]interface{} {
	s3URL, err := k8scommon.BuildMinIOS3URL(minioAccess)
	if err != nil {
		// Validation runs before installer execution, so this fallback is defensive only.
		s3URL = fmt.Sprintf("http://%s", strings.TrimSuffix(minioAccess.Endpoint, "/"))
	}
	bucketName := k8scommon.DefaultMinIOBucket(minioAccess, DefaultVeleroBucket)
	if volumeBackupMode == "" {
		volumeBackupMode = veleromodel.VolumeBackupModeFilesystem
	}

	snapshotsEnabled := false
	deployNodeAgent := true
	defaultVolumesToFsBackup := true
	features := ""

	if volumeBackupMode == veleromodel.VolumeBackupModeSnapshot {
		snapshotsEnabled = true
		deployNodeAgent = false
		defaultVolumesToFsBackup = false
		features = "EnableCSI"
	}

	return map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "docker.io/velero/velero",
			"tag":        veleroImageTag,
		},
		"credentials": map[string]interface{}{
			"useSecret":      true,
			"existingSecret": "cloud-credentials",
		},
		"snapshotsEnabled":         snapshotsEnabled,
		"deployNodeAgent":          deployNodeAgent,
		"defaultVolumesToFsBackup": defaultVolumesToFsBackup,
		"configuration": map[string]interface{}{
			"features": features,
			"backupStorageLocation": []interface{}{
				map[string]interface{}{
					"name":     "minio",
					"provider": "aws",
					"bucket":   bucketName,
					"config": map[string]interface{}{
						"region":           "minio",
						"s3Url":            s3URL,
						"s3ForcePathStyle": "true",
					},
					"credential": map[string]interface{}{
						"name": "cloud-credentials",
						"key":  "cloud",
					},
				},
			},
		},
		"initContainers": []interface{}{
			map[string]interface{}{
				"name":            "velero-plugin-for-aws",
				"image":           "docker.io/velero/velero-plugin-for-aws:v1.13.0",
				"imagePullPolicy": "IfNotPresent",
				"volumeMounts": []interface{}{
					map[string]interface{}{
						"name":      "plugins",
						"mountPath": "/target",
					},
				},
			},
		},
	}
}

func downloadChart(chartURL string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "velero-chart-*")
	if err != nil {
		return "", err
	}

	res, err := http.Get(chartURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode >= 300 {
		return "", fmt.Errorf("failed to download chart: %s", res.Status)
	}

	chartPath := filepath.Join(tmpDir, "velero.tgz")
	fp, err := os.Create(chartPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = fp.Close()
	}()

	_, err = io.Copy(fp, res.Body)
	if err != nil {
		return "", err
	}

	return chartPath, nil
}

func (v *VeleroInstaller) waitForDeploymentReady(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	timeout := time.Now().Add(10 * time.Minute)
	for time.Now().Before(timeout) {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, "velero", metav1.GetOptions{})
		if err == nil && deployment.Status.ReadyReplicas >= 1 {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for velero deployment readiness")
}

func (v *VeleroInstaller) ensureBackupStorageLocation(ctx context.Context, controllerClient ctrlclient.Client, namespace string, minioAccess *commonmodel.MinIOAccess, force bool) error {
	s3URL, err := k8scommon.BuildMinIOS3URL(minioAccess)
	if err != nil {
		return err
	}
	bucketName := k8scommon.DefaultMinIOBucket(minioAccess, DefaultVeleroBucket)

	key := ctrlclient.ObjectKey{Namespace: namespace, Name: "minio"}
	existing := &velerov1.BackupStorageLocation{}
	err = controllerClient.Get(ctx, key, existing)
	if err == nil && !force {
		return nil
	}
	if err == nil && force {
		if delErr := controllerClient.Delete(ctx, existing); delErr != nil {
			return delErr
		}
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	bsl := &velerov1.BackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: namespace,
		},
		Spec: velerov1.BackupStorageLocationSpec{
			Provider: "aws",
			StorageType: velerov1.StorageType{
				ObjectStorage: &velerov1.ObjectStorageLocation{
					Bucket: bucketName,
					Prefix: "backups",
				},
			},
			Config: map[string]string{
				"region":           "minio",
				"s3Url":            s3URL,
				"s3ForcePathStyle": "true",
			},
			Credential: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{Name: "cloud-credentials"},
				Key:                  "cloud",
			},
			Default: true,
		},
	}

	return controllerClient.Create(ctx, bsl)
}
