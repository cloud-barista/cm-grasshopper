package installer

import (
	"context"
	"fmt"
	"io"
	"maps"
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

	veleroBSLName               = "default"
	veleroCredentialsSecretName = "cloud-credentials"
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

func (v *VeleroInstaller) Install(ctx context.Context, cluster *commonmodel.ClusterAccess, s3Access *commonmodel.S3Access, force bool, volumeBackupMode string) (*InstallResult, error) {
	start := time.Now()
	namespace := k8scommon.DefaultNamespace(cluster, DefaultVeleroNamespace)
	bucketName := k8scommon.DefaultS3Bucket(s3Access, DefaultVeleroBucket)
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

	s3Client, err := k8sclient.NewS3Client(s3Access)
	if err != nil {
		return nil, err
	}

	if err := k8sclient.EnsureS3Bucket(ctx, s3Client, bucketName); err != nil {
		return nil, fmt.Errorf("failed to ensure s3 bucket %q: %w", bucketName, err)
	}

	if err := v.ensureSecret(ctx, clientset, namespace, s3Access, force); err != nil {
		return nil, err
	}

	actionConfig, err := k8sclient.NewHelmActionConfig(&commonmodel.ClusterAccess{
		Kubeconfig: cluster.Kubeconfig,
		Namespace:  namespace,
		Tumblebug:  cluster.Tumblebug,
	})
	if err != nil {
		return nil, err
	}

	if err := v.installOrUpgradeChart(actionConfig, namespace, s3Access, force, volumeBackupMode); err != nil {
		return nil, err
	}

	if err := v.waitForDeploymentReady(ctx, clientset, namespace); err != nil {
		return nil, err
	}

	if err := v.ensureBackupStorageLocation(ctx, controllerClient, namespace, s3Access, force); err != nil {
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

func (v *VeleroInstaller) ensureSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace string, s3Access *commonmodel.S3Access, force bool) error {
	desired := buildCloudCredentials(s3Access)
	secrets := clientset.CoreV1().Secrets(namespace)

	if force {
		_ = secrets.Delete(ctx, veleroCredentialsSecretName, metav1.DeleteOptions{})
	}

	existing, err := secrets.Get(ctx, veleroCredentialsSecretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		_, err = secrets.Create(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroCredentialsSecretName,
				Namespace: namespace,
			},
			StringData: map[string]string{"cloud": desired},
		}, metav1.CreateOptions{})

		return err
	}

	if !cloudCredentialsChanged(existing, desired) {
		return nil
	}

	updated := existing.DeepCopy()
	updated.StringData = map[string]string{"cloud": desired}
	delete(updated.Data, "cloud")
	_, err = secrets.Update(ctx, updated, metav1.UpdateOptions{})

	return err
}

// buildCloudCredentials renders the AWS-style credentials file velero's aws plugin reads.
func buildCloudCredentials(s3Access *commonmodel.S3Access) string {
	return fmt.Sprintf(`[default]
aws_access_key_id=%s
aws_secret_access_key=%s
region=%s
`, s3Access.AccessKey, s3Access.SecretKey, k8scommon.DefaultS3Region(s3Access))
}

func toValuesMap(config map[string]string) map[string]interface{} {
	values := make(map[string]interface{}, len(config))
	for k, v := range config {
		values[k] = v
	}

	return values
}

// buildBSLConfig renders the BackupStorageLocation spec.config block. Both the Helm values
// and the controller-runtime path go through here so the two cannot drift apart.
func buildBSLConfig(s3Access *commonmodel.S3Access) (map[string]string, error) {
	s3URL, err := k8scommon.BuildS3URL(s3Access)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"region":           k8scommon.DefaultS3Region(s3Access),
		"s3Url":            s3URL,
		"s3ForcePathStyle": "true",
	}, nil
}

// buildBSLSpec renders the BackupStorageLocation the install should converge on.
func buildBSLSpec(s3Access *commonmodel.S3Access) (velerov1.BackupStorageLocationSpec, error) {
	config, err := buildBSLConfig(s3Access)
	if err != nil {
		return velerov1.BackupStorageLocationSpec{}, err
	}

	return velerov1.BackupStorageLocationSpec{
		Provider: "aws",
		StorageType: velerov1.StorageType{
			ObjectStorage: &velerov1.ObjectStorageLocation{
				Bucket: k8scommon.DefaultS3Bucket(s3Access, DefaultVeleroBucket),
				Prefix: k8scommon.DefaultS3Prefix(s3Access),
			},
		},
		Config: config,
		Credential: &v1.SecretKeySelector{
			LocalObjectReference: v1.LocalObjectReference{Name: veleroCredentialsSecretName},
			Key:                  "cloud",
		},
		Default: true,
	}, nil
}

func bslNeedsUpdate(existing *velerov1.BackupStorageLocation, desired velerov1.BackupStorageLocationSpec) bool {
	if existing == nil {
		return true
	}

	current := existing.Spec
	if current.Provider != desired.Provider || current.Default != desired.Default {
		return true
	}
	if current.ObjectStorage == nil || desired.ObjectStorage == nil {
		return current.ObjectStorage != desired.ObjectStorage
	}
	if current.ObjectStorage.Bucket != desired.ObjectStorage.Bucket ||
		current.ObjectStorage.Prefix != desired.ObjectStorage.Prefix {
		return true
	}

	return !maps.Equal(current.Config, desired.Config)
}

func cloudCredentialsChanged(existing *v1.Secret, desired string) bool {
	if existing == nil {
		return true
	}

	return string(existing.Data["cloud"]) != desired
}

func (v *VeleroInstaller) installOrUpgradeChart(actionConfig *action.Configuration, namespace string, s3Access *commonmodel.S3Access, force bool, volumeBackupMode string) error {
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

	values := buildVeleroValues(s3Access, volumeBackupMode)

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

func buildVeleroValues(s3Access *commonmodel.S3Access, volumeBackupMode string) map[string]interface{} {
	bslConfig, err := buildBSLConfig(s3Access)
	if err != nil {
		// Validation runs before installer execution, so this fallback is defensive only.
		bslConfig = map[string]string{
			"region":           k8scommon.DefaultS3Region(s3Access),
			"s3Url":            fmt.Sprintf("http://%s", strings.TrimSuffix(s3Access.Endpoint, "/")),
			"s3ForcePathStyle": "true",
		}
	}
	bucketName := k8scommon.DefaultS3Bucket(s3Access, DefaultVeleroBucket)
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
					"name":     veleroBSLName,
					"provider": "aws",
					"bucket":   bucketName,
					"prefix":   k8scommon.DefaultS3Prefix(s3Access),
					"config":   toValuesMap(bslConfig),
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

// ensureBackupStorageLocation converges the BackupStorageLocation on buildBSLSpec. The Helm
// chart renders the same location, so an install that skipped an update here used to leave
// whichever spec happened to be applied first - which is how source and target ended up
// pointing at different bucket prefixes.
func (v *VeleroInstaller) ensureBackupStorageLocation(ctx context.Context, controllerClient ctrlclient.Client, namespace string, s3Access *commonmodel.S3Access, force bool) error {
	desired, err := buildBSLSpec(s3Access)
	if err != nil {
		return err
	}

	key := ctrlclient.ObjectKey{Namespace: namespace, Name: veleroBSLName}
	existing := &velerov1.BackupStorageLocation{}
	err = controllerClient.Get(ctx, key, existing)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err == nil && force {
		if delErr := controllerClient.Delete(ctx, existing); delErr != nil {
			return delErr
		}
		err = apierrors.NewNotFound(velerov1.Resource("backupstoragelocations"), veleroBSLName)
	}

	if apierrors.IsNotFound(err) {
		return controllerClient.Create(ctx, &velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      veleroBSLName,
				Namespace: namespace,
			},
			Spec: desired,
		})
	}

	if !bslNeedsUpdate(existing, desired) {
		return nil
	}

	updated := existing.DeepCopy()
	updated.Spec = desired

	return controllerClient.Update(ctx, updated)
}
