package velero

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	joblib "github.com/cloud-barista/cm-grasshopper/lib/job"
	k8sclient "github.com/cloud-barista/cm-grasshopper/lib/k8s/client"
	k8scommon "github.com/cloud-barista/cm-grasshopper/lib/k8s/common"
	k8sinstaller "github.com/cloud-barista/cm-grasshopper/lib/k8s/installer"
	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
	veleromodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/velero"
	"github.com/google/uuid"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metautils "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	installer *k8sinstaller.VeleroInstaller
}

const (
	storageClassConfigMapName = "cm-grasshopper-change-storage-class"
)

func NewService() *Service {
	return &Service{
		installer: k8sinstaller.NewVeleroInstaller(),
	}
}

func updateJobProgressSafe(jobID string, progress int, message string) {
	if joblib.DefaultManager == nil {
		return
	}
	_ = joblib.DefaultManager.UpdateJobStatus(jobID, joblib.StatusProcessing, progress, message)
}

func formatMetaTime(t metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatMetaTimePtr(t *metav1.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *Service) HealthCheck(ctx context.Context, cluster *commonmodel.ClusterAccess) (*veleromodel.HealthResponse, error) {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	list := &velerov1.BackupList{}
	if err := controllerClient.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		return nil, err
	}

	response := &veleromodel.HealthResponse{
		Status:    "ok",
		Cluster:   strings.TrimSpace(cluster.Name),
		Namespace: namespace,
	}

	bsl := &velerov1.BackupStorageLocation{}
	if err := controllerClient.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: "minio"}, bsl); err == nil {
		summary := &veleromodel.BackupStorageLocationHealth{
			Name:  bsl.Name,
			Phase: string(bsl.Status.Phase),
		}
		if bsl.Status.Message != "" {
			summary.Message = bsl.Status.Message
		}
		if !bsl.Status.LastValidationTime.IsZero() {
			summary.LastValidationTime = formatMetaTimePtr(bsl.Status.LastValidationTime)
		}
		response.BackupStorageLocation = summary
		if bsl.Status.Phase != velerov1.BackupStorageLocationPhaseAvailable {
			response.Status = "degraded"
		}
	}

	return response, nil
}

func (s *Service) InstallAsync(clusterRole string, cluster *commonmodel.ClusterAccess, minioAccess *commonmodel.MinIOAccess, force bool, volumeBackupMode string) (*joblib.Info, error) {
	if joblib.DefaultManager == nil {
		return nil, fmt.Errorf("job manager is not initialized")
	}
	if volumeBackupMode == "" {
		volumeBackupMode = veleromodel.VolumeBackupModeFilesystem
	}

	metadata := map[string]interface{}{
		"clusterRole": clusterRole,
		"cluster": map[string]interface{}{
			"name":                  cluster.Name,
			"namespace":             k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace),
			"kubeconfigFingerprint": joblib.KubeconfigFingerprint(cluster.Kubeconfig),
		},
		"minio": map[string]interface{}{
			"endpoint":        minioAccess.Endpoint,
			"bucket":          k8scommon.DefaultMinIOBucket(minioAccess, k8sinstaller.DefaultVeleroBucket),
			"accessKeyMasked": joblib.MaskSecret(minioAccess.AccessKey),
		},
		"force":            force,
		"volumeBackupMode": volumeBackupMode,
	}

	job, err := joblib.DefaultManager.CreateJob("velero_install", "velero", clusterRole, metadata)
	if err != nil {
		return nil, err
	}

	joblib.DefaultManager.Submit(func() {
		_ = joblib.DefaultManager.UpdateJobStatus(job.JobID, joblib.StatusProcessing, 10, "Starting Velero installation")
		_ = joblib.DefaultManager.AddJobLog(job.JobID, "Starting Velero installation")

		ctx, cancel := context.WithTimeout(context.Background(), config.GetK8sInstallTimeout())
		defer cancel()

		result, installErr := s.installer.Install(ctx, cluster, minioAccess, force, volumeBackupMode)
		if installErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, installErr)
			return
		}

		_ = joblib.DefaultManager.AddJobLog(job.JobID, fmt.Sprintf("Velero installed in namespace %s", result.Namespace))
		_ = joblib.DefaultManager.CompleteJob(job.JobID, result.Message)
	})

	return job, nil
}

func (s *Service) Precheck(ctx context.Context, sourceCluster, targetCluster *commonmodel.ClusterAccess, minioAccess *commonmodel.MinIOAccess, spec veleromodel.MigrationPrecheckSpec) (*veleromodel.PrecheckResponse, error) {
	sourceHealth, err := s.HealthCheck(ctx, sourceCluster)
	if err != nil {
		return nil, fmt.Errorf("source cluster velero health check failed: %w", err)
	}
	targetHealth, err := s.HealthCheck(ctx, targetCluster)
	if err != nil {
		return nil, fmt.Errorf("target cluster velero health check failed: %w", err)
	}

	sourceClientset, sourceControllerClient, err := k8sclient.NewKubernetesClient(sourceCluster)
	if err != nil {
		return nil, err
	}
	targetClientset, targetControllerClient, err := k8sclient.NewKubernetesClient(targetCluster)
	if err != nil {
		return nil, err
	}

	minioClient, err := k8sclient.NewMinIOClient(minioAccess)
	if err != nil {
		return nil, err
	}
	bucketName := k8scommon.DefaultMinIOBucket(minioAccess, k8sinstaller.DefaultVeleroBucket)

	if err := k8sclient.EnsureMinIOBucket(ctx, minioClient, bucketName); err != nil {
		return nil, fmt.Errorf("failed to ensure minio bucket %q: %w", bucketName, err)
	}

	result := &veleromodel.PrecheckResponse{
		Status: "ready",
		Source: veleromodel.PrecheckClusterSummary{
			Name:      sourceCluster.Name,
			Namespace: k8scommon.DefaultNamespace(sourceCluster, k8sinstaller.DefaultVeleroNamespace),
		},
		Target: veleromodel.PrecheckClusterSummary{
			Name:      targetCluster.Name,
			Namespace: k8scommon.DefaultNamespace(targetCluster, k8sinstaller.DefaultVeleroNamespace),
		},
		Storage: veleromodel.PrecheckStorageSummary{
			Endpoint: minioAccess.Endpoint,
			Bucket:   bucketName,
		},
		Warnings: []string{},
		Errors:   []string{},
	}

	if sourceHealth.BackupStorageLocation != nil {
		result.Source.BackupStorageLocation = sourceHealth.BackupStorageLocation
		if sourceHealth.BackupStorageLocation.Phase != string(velerov1.BackupStorageLocationPhaseAvailable) {
			message := sourceHealth.BackupStorageLocation.Message
			if message == "" {
				message = fmt.Sprintf("source cluster BackupStorageLocation %q is not available", sourceHealth.BackupStorageLocation.Name)
			}
			result.Errors = append(result.Errors, message)
		}
	}
	if targetHealth.BackupStorageLocation != nil {
		result.Target.BackupStorageLocation = targetHealth.BackupStorageLocation
		if targetHealth.BackupStorageLocation.Phase != string(velerov1.BackupStorageLocationPhaseAvailable) {
			message := targetHealth.BackupStorageLocation.Message
			if message == "" {
				message = fmt.Sprintf("target cluster BackupStorageLocation %q is not available", targetHealth.BackupStorageLocation.Name)
			}
			result.Errors = append(result.Errors, message)
		}
	}

	sourceNamespaces := buildSourceNamespaces(spec.SourceNamespace, spec.IncludedNamespaces)
	if len(sourceNamespaces) == 0 {
		result.Warnings = append(result.Warnings, "namespace not specified; namespace-specific precheck was skipped")
	} else {
		result.Source.SourceNamespaces = sourceNamespaces
		for _, namespace := range sourceNamespaces {
			if _, err := sourceClientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{}); err != nil {
				if apierrors.IsNotFound(err) {
					result.Errors = append(result.Errors, fmt.Sprintf("source namespace %q does not exist", namespace))
					continue
				}
				return nil, err
			}
		}
	}

	namespaceMapping := buildNamespaceMapping(spec.SourceNamespace, spec.TargetNamespace, sourceNamespaces, spec.NamespaceMapping)
	if spec.TargetNamespace != "" && len(namespaceMapping) == 0 {
		result.Errors = append(result.Errors, "targetNamespace requires namespace or a single included namespace")
	}

	if len(namespaceMapping) == 0 && len(sourceNamespaces) == 1 && spec.TargetNamespace == "" {
		targetNamespace := sourceNamespaces[0]
		namespaceMapping = map[string]string{targetNamespace: targetNamespace}
	}

	targetNamespaceStatuses := map[string]string{}
	for _, targetNamespace := range uniqueMapValues(namespaceMapping) {
		_, err := targetClientset.CoreV1().Namespaces().Get(ctx, targetNamespace, metav1.GetOptions{})
		switch {
		case err == nil:
			targetNamespaceStatuses[targetNamespace] = "exists"
			result.Warnings = append(result.Warnings, fmt.Sprintf("target namespace %q already exists; review for overlapping resources before restore", targetNamespace))
		case apierrors.IsNotFound(err):
			targetNamespaceStatuses[targetNamespace] = "will-be-created"
			result.Warnings = append(result.Warnings, fmt.Sprintf("target namespace %q does not exist and will be created during restore", targetNamespace))
		default:
			return nil, err
		}
	}
	if len(targetNamespaceStatuses) > 0 {
		result.Target.NamespaceStatus = targetNamespaceStatuses
	}

	sourceStorageClasses, err := collectSourceStorageClasses(ctx, sourceClientset, sourceNamespaces)
	if err != nil {
		return nil, err
	}
	targetStorageClasses, err := collectTargetStorageClasses(ctx, targetClientset)
	if err != nil {
		return nil, err
	}
	if len(sourceStorageClasses) > 0 {
		result.Source.StorageClasses = sourceStorageClasses
	}
	if len(targetStorageClasses) > 0 {
		result.Target.StorageClasses = sortedKeys(targetStorageClasses)
	}

	missingMappings := []string{}
	missingTargets := []string{}
	usedMappings := map[string]string{}
	unusedMappings := map[string]string{}
	for _, sourceStorageClass := range sourceStorageClasses {
		mappedTargetStorageClass, mapped := spec.StorageClassMappings[sourceStorageClass]
		if mapped {
			usedMappings[sourceStorageClass] = mappedTargetStorageClass
			if _, exists := targetStorageClasses[mappedTargetStorageClass]; !exists {
				missingTargets = append(missingTargets, fmt.Sprintf("%s->%s", sourceStorageClass, mappedTargetStorageClass))
			}
			continue
		}

		if _, exists := targetStorageClasses[sourceStorageClass]; !exists {
			missingMappings = append(missingMappings, sourceStorageClass)
		}
	}
	if len(missingMappings) > 0 {
		sort.Strings(missingMappings)
		result.Warnings = append(result.Warnings, fmt.Sprintf("storageClassMappings recommended for source storage classes not present on target: %s", strings.Join(missingMappings, ", ")))
	}
	if len(missingTargets) > 0 {
		sort.Strings(missingTargets)
		result.Errors = append(result.Errors, fmt.Sprintf("mapped target storage classes do not exist on target cluster: %s", strings.Join(missingTargets, ", ")))
	}
	for sourceStorageClass, targetStorageClass := range spec.StorageClassMappings {
		if _, exists := usedMappings[sourceStorageClass]; exists {
			continue
		}
		unusedMappings[sourceStorageClass] = targetStorageClass
	}
	if len(usedMappings) > 0 || len(unusedMappings) > 0 || len(missingMappings) > 0 {
		recommendation := &veleromodel.StorageClassRecommendation{
			MappingRequired: len(missingMappings) > 0,
			UsedMappings:    usedMappings,
			UnusedMappings:  unusedMappings,
			MissingMappings: missingMappings,
		}
		if len(missingMappings) > 0 {
			suggestedMappings := map[string]string{}
			for _, sourceStorageClass := range missingMappings {
				for targetStorageClass := range targetStorageClasses {
					suggestedMappings[sourceStorageClass] = targetStorageClass
					break
				}
			}
			recommendation.SuggestedMappings = suggestedMappings
		}
		result.Source.StorageClassRecommendation = recommendation
	}

	compatibility, compatibilityWarnings, compatibilityErrors, err := assessSourceVolumeBackupCompatibility(ctx, sourceClientset, sourceNamespaces, spec.VolumeBackupMode)
	if err != nil {
		return nil, err
	}
	if compatibility != nil {
		supportedModes := []string{}
		if compatibility.FilesystemBackupReady {
			supportedModes = append(supportedModes, veleromodel.VolumeBackupModeFilesystem)
		}
		snapshotSupport, snapshotWarnings, snapshotErrors, snapshotErr := assessSnapshotCompatibility(ctx, sourceClientset, targetClientset, sourceControllerClient, targetControllerClient, sourceNamespaces, spec.StorageClassMappings)
		if snapshotErr != nil {
			return nil, snapshotErr
		}
		if snapshotSupport != nil {
			compatibility.SnapshotSupport = snapshotSupport
			if snapshotSupport.SnapshotReady {
				supportedModes = append(supportedModes, veleromodel.VolumeBackupModeSnapshot)
			}
			result.Warnings = append(result.Warnings, snapshotWarnings...)
			result.Errors = append(result.Errors, snapshotErrors...)
		}
		compatibility.SupportedVolumeBackupModes = supportedModes
		switch {
		case containsString(supportedModes, veleromodel.VolumeBackupModeSnapshot):
			compatibility.RecommendedVolumeBackupMode = veleromodel.VolumeBackupModeSnapshot
			compatibility.RecommendedAction = "snapshot backup is available and preferred for the inspected source volumes"
		case containsString(supportedModes, veleromodel.VolumeBackupModeFilesystem):
			compatibility.RecommendedVolumeBackupMode = veleromodel.VolumeBackupModeFilesystem
			compatibility.RecommendedAction = "filesystem backup is acceptable for the inspected source volumes"
		default:
			compatibility.RecommendedVolumeBackupMode = ""
			compatibility.RecommendedAction = "change the source storage class to a non-hostPath-backed volume or migrate PVC data outside Velero"
		}
		result.Source.VolumeBackupCompatibility = compatibility
	}
	result.Warnings = append(result.Warnings, compatibilityWarnings...)
	result.Errors = append(result.Errors, compatibilityErrors...)

	if len(result.Errors) > 0 {
		result.Status = "not_ready"
	} else if len(result.Warnings) > 0 {
		result.Status = "ready_with_warnings"
	}

	return result, nil
}

func CompactPrecheckResponse(response *veleromodel.PrecheckResponse) *veleromodel.PrecheckCompactResponse {
	if response == nil {
		return nil
	}

	recommendedMode := ""
	summary := veleromodel.PrecheckSummary{
		SourceNamespaces: response.Source.SourceNamespaces,
		TargetNamespaces: response.Target.NamespaceStatus,
	}

	if response.Source.BackupStorageLocation != nil {
		summary.BackupStorageLocationReady = response.Source.BackupStorageLocation.Phase == string(velerov1.BackupStorageLocationPhaseAvailable)
	}
	if recommendation := response.Source.StorageClassRecommendation; recommendation != nil {
		summary.StorageClassMappingRequired = recommendation.MappingRequired
	}
	if compatibility := response.Source.VolumeBackupCompatibility; compatibility != nil {
		recommendedMode = compatibility.RecommendedVolumeBackupMode
		summary.RecommendedVolumeBackupMode = recommendedMode
	}

	return &veleromodel.PrecheckCompactResponse{
		Status:   response.Status,
		Summary:  summary,
		Warnings: filterCompactPrecheckWarnings(response.Warnings, recommendedMode),
		Errors:   response.Errors,
	}
}

func filterCompactPrecheckWarnings(warnings []string, recommendedMode string) []string {
	if len(warnings) == 0 {
		return warnings
	}

	filtered := make([]string, 0, len(warnings))
	seen := map[string]struct{}{}

	for _, warning := range warnings {
		if recommendedMode == veleromodel.VolumeBackupModeFilesystem && isSnapshotOnlyPrecheckWarning(warning) {
			continue
		}
		if _, exists := seen[warning]; exists {
			continue
		}
		seen[warning] = struct{}{}
		filtered = append(filtered, warning)
	}

	return filtered
}

func isSnapshotOnlyPrecheckWarning(warning string) bool {
	lower := strings.ToLower(strings.TrimSpace(warning))
	if lower == "" {
		return false
	}

	return strings.Contains(lower, "volumesnapshotclass") ||
		strings.Contains(lower, "snapshot backup is not available") ||
		strings.Contains(lower, "snapshot restore is not available") ||
		strings.Contains(lower, "snapshot backup check skipped")
}

func (s *Service) ExecuteMigrationAsync(sourceCluster, targetCluster *commonmodel.ClusterAccess, minioAccess *commonmodel.MinIOAccess, spec veleromodel.MigrationExecuteSpec) (*joblib.Info, error) {
	if joblib.DefaultManager == nil {
		return nil, fmt.Errorf("job manager is not initialized")
	}

	backupName := spec.BackupName
	if backupName == "" {
		backupName = fmt.Sprintf("backup-%d", time.Now().Unix())
	}
	restoreName := spec.RestoreName
	if restoreName == "" {
		restoreName = fmt.Sprintf("restore-%d", time.Now().Unix())
	}

	metadata := map[string]interface{}{
		"sourceCluster": map[string]interface{}{
			"name":                  sourceCluster.Name,
			"namespace":             k8scommon.DefaultNamespace(sourceCluster, k8sinstaller.DefaultVeleroNamespace),
			"kubeconfigFingerprint": joblib.KubeconfigFingerprint(sourceCluster.Kubeconfig),
		},
		"targetCluster": map[string]interface{}{
			"name":                  targetCluster.Name,
			"namespace":             k8scommon.DefaultNamespace(targetCluster, k8sinstaller.DefaultVeleroNamespace),
			"kubeconfigFingerprint": joblib.KubeconfigFingerprint(targetCluster.Kubeconfig),
		},
		"minio": map[string]interface{}{
			"endpoint":        minioAccess.Endpoint,
			"bucket":          k8scommon.DefaultMinIOBucket(minioAccess, k8sinstaller.DefaultVeleroBucket),
			"accessKeyMasked": joblib.MaskSecret(minioAccess.AccessKey),
		},
		"backupName":      backupName,
		"restoreName":     restoreName,
		"targetNamespace": spec.TargetNamespace,
	}

	job, err := joblib.DefaultManager.CreateJob("velero_migration_execute", "migration", restoreName, metadata)
	if err != nil {
		return nil, err
	}

	joblib.DefaultManager.Submit(func() {
		updateJobProgressSafe(job.JobID, 5, "Starting migration precheck")
		_ = joblib.DefaultManager.AddJobLog(job.JobID, "Starting migration precheck")

		ctx, cancel := context.WithTimeout(context.Background(), config.GetK8sBackupTimeout()+config.GetK8sRestoreTimeout())
		defer cancel()

		precheckSpec := veleromodel.MigrationPrecheckSpec{
			BackupName:              backupName,
			RestoreName:             restoreName,
			SourceNamespace:         spec.SourceNamespace,
			TargetNamespace:         spec.TargetNamespace,
			IncludedNamespaces:      spec.IncludedNamespaces,
			ExcludedNamespaces:      spec.ExcludedNamespaces,
			IncludedResources:       spec.IncludedResources,
			ExcludedResources:       spec.ExcludedResources,
			NamespaceMapping:        spec.NamespaceMapping,
			StorageClassMappings:    spec.StorageClassMappings,
			IncludeClusterResources: spec.IncludeClusterResources,
			VolumeBackupMode:        spec.VolumeBackupMode,
		}
		precheckResult, precheckErr := s.Precheck(ctx, sourceCluster, targetCluster, minioAccess, precheckSpec)
		if precheckErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, precheckErr)
			return
		}
		for _, warning := range precheckResult.Warnings {
			_ = joblib.DefaultManager.AddJobLogWithLevel(job.JobID, joblib.LogLevelWarn, "Precheck warning: "+warning)
		}
		if precheckResult.Status == "not_ready" {
			if len(precheckResult.Errors) > 0 {
				_ = joblib.DefaultManager.FailJob(job.JobID, fmt.Errorf("migration precheck failed: %s", strings.Join(precheckResult.Errors, "; ")))
				return
			}
			_ = joblib.DefaultManager.FailJob(job.JobID, fmt.Errorf("migration precheck failed"))
			return
		}
		updateJobProgressSafe(job.JobID, 15, fmt.Sprintf("Migration precheck finished with status %s", precheckResult.Status))

		backupSpec := veleromodel.BackupSpec{
			Name:                     backupName,
			SourceNamespace:          spec.SourceNamespace,
			IncludedNamespaces:       spec.IncludedNamespaces,
			ExcludedNamespaces:       spec.ExcludedNamespaces,
			IncludedResources:        spec.IncludedResources,
			ExcludedResources:        spec.ExcludedResources,
			IncludeClusterResources:  spec.IncludeClusterResources,
			VolumeBackupMode:         spec.VolumeBackupMode,
			NameConflictPolicy:       spec.NameConflictPolicy,
			SnapshotVolumes:          spec.SnapshotVolumes,
			DefaultVolumesToFsBackup: spec.DefaultVolumesToFsBackup,
		}

		updateJobProgressSafe(job.JobID, 20, fmt.Sprintf("Creating source backup %s", backupName))
		_ = joblib.DefaultManager.AddJobLog(job.JobID, fmt.Sprintf("Creating source backup %s", backupName))
		backupResult, createErr := s.CreateBackup(ctx, sourceCluster, backupSpec)
		if createErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, createErr)
			return
		}
		if backupResult != nil && backupResult.Name != "" {
			backupName = backupResult.Name
		}
		updateJobProgressSafe(job.JobID, 30, fmt.Sprintf("Source backup %s created; waiting for completion", backupName))

		backup, waitBackupErr := s.waitForBackupCompletion(ctx, sourceCluster, backupName, job.JobID, 40)
		if waitBackupErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, waitBackupErr)
			return
		}
		updateJobProgressSafe(job.JobID, 65, fmt.Sprintf("Backup %s completed with phase %s", backupName, backup.Status.Phase))

		updateJobProgressSafe(job.JobID, 70, fmt.Sprintf("Waiting for backup %s to sync into target cluster", backupName))
		_ = joblib.DefaultManager.AddJobLog(job.JobID, fmt.Sprintf("Waiting for backup %s to sync into target cluster", backupName))
		if waitSyncErr := s.waitForBackupSync(ctx, targetCluster, backupName, job.JobID, 72); waitSyncErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, waitSyncErr)
			return
		}
		updateJobProgressSafe(job.JobID, 75, fmt.Sprintf("Backup %s is available on target cluster", backupName))

		restoreSpec := veleromodel.RestoreSpec{
			Name:                    restoreName,
			BackupName:              backupName,
			SourceNamespace:         spec.SourceNamespace,
			TargetNamespace:         spec.TargetNamespace,
			IncludedNamespaces:      spec.IncludedNamespaces,
			ExcludedNamespaces:      spec.ExcludedNamespaces,
			IncludedResources:       spec.IncludedResources,
			ExcludedResources:       spec.ExcludedResources,
			NamespaceMapping:        spec.NamespaceMapping,
			StorageClassMappings:    spec.StorageClassMappings,
			IncludeClusterResources: spec.IncludeClusterResources,
			ExistingResourcePolicy:  spec.ExistingResourcePolicy,
			RestorePVs:              spec.RestorePVs,
		}

		updateJobProgressSafe(job.JobID, 80, fmt.Sprintf("Creating target restore %s", restoreName))
		_ = joblib.DefaultManager.AddJobLog(job.JobID, fmt.Sprintf("Creating target restore %s", restoreName))
		if _, createErr := s.CreateRestore(ctx, targetCluster, restoreSpec); createErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, createErr)
			return
		}
		updateJobProgressSafe(job.JobID, 85, fmt.Sprintf("Target restore %s created; waiting for completion", restoreName))

		restore, waitRestoreErr := s.waitForRestoreCompletion(ctx, targetCluster, restoreName, job.JobID, 90)
		if waitRestoreErr != nil {
			_ = joblib.DefaultManager.FailJob(job.JobID, waitRestoreErr)
			return
		}

		finalMessage := fmt.Sprintf("Migration completed: backup=%s restore=%s phase=%s", backupName, restoreName, restore.Status.Phase)
		_ = joblib.DefaultManager.AddJobLog(job.JobID, finalMessage)
		_ = joblib.DefaultManager.CompleteJob(job.JobID, finalMessage)
	})

	return job, nil
}

func (s *Service) ListBackups(ctx context.Context, cluster *commonmodel.ClusterAccess) ([]*veleromodel.BackupResponse, error) {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	list := &velerov1.BackupList{}
	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	if err := controllerClient.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		return nil, err
	}

	result := make([]*veleromodel.BackupResponse, 0, len(list.Items))
	for i := range list.Items {
		list.Items[i].ManagedFields = nil
		result = append(result, backupResponse(&list.Items[i]))
	}
	return result, nil
}

func (s *Service) GetBackup(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*veleromodel.BackupResponse, error) {
	backup, err := s.getBackupResource(ctx, cluster, name)
	if err != nil {
		return nil, err
	}
	return backupResponse(backup), nil
}

func (s *Service) getBackupResource(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*velerov1.Backup, error) {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	backup := &velerov1.Backup{}
	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	if err := controllerClient.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: name}, backup); err != nil {
		return nil, err
	}
	backup.ManagedFields = nil
	return backup, nil
}

func (s *Service) CreateBackup(ctx context.Context, cluster *commonmodel.ClusterAccess, spec veleromodel.BackupSpec) (*veleromodel.BackupResponse, error) {
	clientset, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	spec, err = normalizeBackupSpec(spec)
	if err != nil {
		return nil, err
	}

	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	requestedName := spec.Name
	actualName := spec.Name

	for attempt := 0; attempt < 5; attempt++ {
		backup := &velerov1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      actualName,
				Namespace: namespace,
			},
			Spec: velerov1.BackupSpec{
				IncludedNamespaces:       spec.IncludedNamespaces,
				ExcludedNamespaces:       spec.ExcludedNamespaces,
				IncludedResources:        spec.IncludedResources,
				ExcludedResources:        spec.ExcludedResources,
				IncludeClusterResources:  spec.IncludeClusterResources,
				SnapshotVolumes:          &spec.SnapshotVolumes,
				StorageLocation:          "minio",
				DefaultVolumesToFsBackup: &spec.DefaultVolumesToFsBackup,
			},
		}

		if err := controllerClient.Create(ctx, backup); err != nil {
			if apierrors.IsAlreadyExists(err) {
				if spec.NameConflictPolicy == veleromodel.NameConflictPolicyFail {
					return nil, fmt.Errorf("backup %q already exists", actualName)
				}
				actualName = uniqueBackupName(requestedName)
				continue
			}
			return nil, err
		}

		response := backupResponse(backup)
		compatibility, compatibilityWarnings, compatibilityErrors, assessErr := assessSourceVolumeBackupCompatibility(ctx, clientset, backup.Spec.IncludedNamespaces, spec.VolumeBackupMode)
		if assessErr != nil {
			return nil, assessErr
		}
		if compatibility != nil {
			response.VolumeBackupCompatibility = compatibility
		}
		if len(compatibilityWarnings) > 0 {
			response.CompatibilityWarnings = compatibilityWarnings
		}
		if len(compatibilityErrors) > 0 {
			response.CompatibilityErrors = compatibilityErrors
		}
		response.RequestedName = requestedName
		response.NameAdjusted = requestedName != actualName
		return response, nil
	}

	return nil, fmt.Errorf("failed to allocate unique backup name for %q", requestedName)
}

func backupResponse(backup *velerov1.Backup) *veleromodel.BackupResponse {
	response := &veleromodel.BackupResponse{
		Name:      backup.Name,
		Namespace: backup.Namespace,
		Phase:     string(backup.Status.Phase),
		Warnings:  backup.Status.Warnings,
		Errors:    backup.Status.Errors,
		CreatedAt: formatMetaTime(backup.CreationTimestamp),
		Started:   formatMetaTimePtr(backup.Status.StartTimestamp),
		Completed: formatMetaTimePtr(backup.Status.CompletionTimestamp),
		BackupMode: deriveBackupMode(
			backup.Spec.DefaultVolumesToFsBackup,
			backup.Spec.SnapshotVolumes,
		),
		Storage: backup.Spec.StorageLocation,
		TTL:     backup.Spec.TTL.Duration.String(),
	}

	if len(backup.Spec.IncludedNamespaces) > 0 {
		response.IncludedNamespaces = backup.Spec.IncludedNamespaces
	}
	if len(backup.Spec.ExcludedNamespaces) > 0 {
		response.ExcludedNamespaces = backup.Spec.ExcludedNamespaces
	}
	if len(backup.Spec.IncludedResources) > 0 {
		response.IncludedResources = backup.Spec.IncludedResources
	}
	if len(backup.Spec.ExcludedResources) > 0 {
		response.ExcludedResources = backup.Spec.ExcludedResources
	}
	response.IncludeClusterResources = backup.Spec.IncludeClusterResources
	response.DefaultVolumesToFsBackup = backup.Spec.DefaultVolumesToFsBackup
	response.SnapshotVolumes = backup.Spec.SnapshotVolumes

	return response
}

func (s *Service) DeleteBackup(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) error {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return err
	}

	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	return controllerClient.Delete(ctx, &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
}

func (s *Service) ValidateBackup(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*veleromodel.BackupResponse, error) {
	backup, err := s.getBackupResource(ctx, cluster, name)
	if err != nil {
		return nil, err
	}
	response := backupResponse(backup)
	clientset, _, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}
	compatibility, compatibilityWarnings, compatibilityErrors, assessErr := assessSourceVolumeBackupCompatibility(ctx, clientset, backup.Spec.IncludedNamespaces, deriveBackupMode(backup.Spec.DefaultVolumesToFsBackup, backup.Spec.SnapshotVolumes))
	if assessErr != nil {
		return nil, assessErr
	}
	if compatibility != nil {
		response.VolumeBackupCompatibility = compatibility
	}
	if len(compatibilityWarnings) > 0 {
		response.CompatibilityWarnings = compatibilityWarnings
	}
	if len(compatibilityErrors) > 0 {
		response.CompatibilityErrors = compatibilityErrors
	}
	return response, nil
}

func (s *Service) ListRestores(ctx context.Context, cluster *commonmodel.ClusterAccess) ([]*veleromodel.RestoreResponse, error) {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	list := &velerov1.RestoreList{}
	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	if err := controllerClient.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		return nil, err
	}

	result := make([]*veleromodel.RestoreResponse, 0, len(list.Items))
	for i := range list.Items {
		list.Items[i].ManagedFields = nil
		result = append(result, restoreResponse(&list.Items[i]))
	}
	return result, nil
}

func (s *Service) GetRestore(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*veleromodel.RestoreResponse, error) {
	restore, err := s.getRestoreResource(ctx, cluster, name)
	if err != nil {
		return nil, err
	}
	return restoreResponse(restore), nil
}

func (s *Service) getRestoreResource(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*velerov1.Restore, error) {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	restore := &velerov1.Restore{}
	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	if err := controllerClient.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: name}, restore); err != nil {
		return nil, err
	}
	restore.ManagedFields = nil
	return restore, nil
}

func restoreResponse(restore *velerov1.Restore) *veleromodel.RestoreResponse {
	response := &veleromodel.RestoreResponse{
		Name:                   restore.Name,
		Namespace:              restore.Namespace,
		Phase:                  string(restore.Status.Phase),
		Warnings:               restore.Status.Warnings,
		Errors:                 restore.Status.Errors,
		ValidationErrors:       restore.Status.ValidationErrors,
		Started:                formatMetaTimePtr(restore.Status.StartTimestamp),
		Completed:              formatMetaTimePtr(restore.Status.CompletionTimestamp),
		CreatedAt:              formatMetaTime(restore.CreationTimestamp),
		BackupName:             restore.Spec.BackupName,
		RestorePVs:             restore.Spec.RestorePVs,
		ExistingResourcePolicy: string(restore.Spec.ExistingResourcePolicy),
	}

	if len(restore.Spec.IncludedNamespaces) > 0 {
		response.IncludedNamespaces = restore.Spec.IncludedNamespaces
	}
	if len(restore.Spec.ExcludedNamespaces) > 0 {
		response.ExcludedNamespaces = restore.Spec.ExcludedNamespaces
	}
	if len(restore.Spec.IncludedResources) > 0 {
		response.IncludedResources = restore.Spec.IncludedResources
	}
	if len(restore.Spec.ExcludedResources) > 0 {
		response.ExcludedResources = restore.Spec.ExcludedResources
	}
	if len(restore.Spec.NamespaceMapping) > 0 {
		response.NamespaceMapping = restore.Spec.NamespaceMapping
	}
	response.IncludeClusterResources = restore.Spec.IncludeClusterResources

	return response
}

func (s *Service) CreateRestore(ctx context.Context, cluster *commonmodel.ClusterAccess, spec veleromodel.RestoreSpec) (*veleromodel.RestoreResponse, error) {
	clientset, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return nil, err
	}

	spec, err = normalizeRestoreSpec(spec)
	if err != nil {
		return nil, err
	}
	if _, err := s.getBackupResource(ctx, cluster, spec.BackupName); err != nil {
		return nil, err
	}

	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	if err := s.ensureStorageClassConfigMap(ctx, clientset, namespace, spec.StorageClassMappings); err != nil {
		return nil, err
	}

	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: namespace,
		},
		Spec: velerov1.RestoreSpec{
			BackupName:              spec.BackupName,
			IncludedNamespaces:      spec.IncludedNamespaces,
			ExcludedNamespaces:      spec.ExcludedNamespaces,
			IncludedResources:       spec.IncludedResources,
			ExcludedResources:       spec.ExcludedResources,
			NamespaceMapping:        spec.NamespaceMapping,
			IncludeClusterResources: spec.IncludeClusterResources,
			ExistingResourcePolicy:  velerov1.PolicyType(spec.ExistingResourcePolicy),
			RestorePVs:              spec.RestorePVs,
		},
	}

	if err := controllerClient.Create(ctx, restore); err != nil {
		return nil, err
	}
	return restoreResponse(restore), nil
}

func normalizeBackupSpec(spec veleromodel.BackupSpec) (veleromodel.BackupSpec, error) {
	if spec.Name == "" {
		spec.Name = uniqueBackupName("backup")
	}
	if spec.SourceNamespace != "" && len(spec.IncludedNamespaces) == 0 {
		spec.IncludedNamespaces = []string{spec.SourceNamespace}
	}
	switch spec.NameConflictPolicy {
	case "", veleromodel.NameConflictPolicyRename:
		spec.NameConflictPolicy = veleromodel.NameConflictPolicyRename
	case veleromodel.NameConflictPolicyFail:
	default:
		return spec, fmt.Errorf("nameConflictPolicy must be one of: rename, fail")
	}

	switch spec.VolumeBackupMode {
	case "", veleromodel.VolumeBackupModeFilesystem:
		spec.DefaultVolumesToFsBackup = true
		spec.SnapshotVolumes = false
		spec.IncludedResources = ensureFilesystemVolumeResources(spec.IncludedResources)
	case veleromodel.VolumeBackupModeSnapshot:
		spec.DefaultVolumesToFsBackup = false
		spec.SnapshotVolumes = true
	default:
		return spec, fmt.Errorf("volumeBackupMode must be one of: filesystem, snapshot")
	}
	return spec, nil
}

func uniqueBackupName(prefix string) string {
	base := strings.TrimSpace(prefix)
	if base == "" {
		base = "backup"
	}
	return fmt.Sprintf("%s-%s-%s", base, time.Now().Format("20060102-150405"), uuid.NewString()[:8])
}

func ensureFilesystemVolumeResources(resources []string) []string {
	if len(resources) == 0 {
		return resources
	}
	if !containsResource(resources, "persistentvolumeclaims") || containsResource(resources, "persistentvolumes") {
		return resources
	}

	result := append([]string{}, resources...)
	result = append(result, "persistentvolumes")
	return result
}

func assessSourceVolumeBackupCompatibility(ctx context.Context, clientset *kubernetes.Clientset, namespaces []string, volumeBackupMode string) (*veleromodel.VolumeBackupCompatibility, []string, []string, error) {
	mode := volumeBackupMode
	if mode == "" {
		mode = veleromodel.VolumeBackupModeFilesystem
	}

	result := &veleromodel.VolumeBackupCompatibility{
		VolumeBackupMode: mode,
	}
	if mode != veleromodel.VolumeBackupModeFilesystem || len(namespaces) == 0 {
		result.SupportedVolumeBackupModes = []string{mode}
		result.RecommendedVolumeBackupMode = mode
		return result, nil, nil, nil
	}

	podWarnings := []string{}
	storageWarnings := []string{}
	compatibilityErrors := []string{}
	unsupportedVolumes := []veleromodel.UnsupportedVolume{}
	storageClasses := map[string]storagev1.StorageClass{}

	for _, namespace := range namespaces {
		podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, nil, nil, err
		}
		for _, pod := range podList.Items {
			for _, volume := range pod.Spec.Volumes {
				if volume.HostPath == nil {
					continue
				}
				podWarnings = append(podWarnings, fmt.Sprintf("pod %s/%s uses direct hostPath volume %q, which is not supported by Velero file system backup", namespace, pod.Name, volume.Name))
				unsupportedVolumes = append(unsupportedVolumes, veleromodel.UnsupportedVolume{
					Type:      "podHostPath",
					Namespace: namespace,
					Pod:       pod.Name,
					Volume:    volume.Name,
					Reason:    "hostPath is not supported by Velero file system backup",
				})
			}
		}

		pvcList, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, nil, nil, err
		}
		for _, pvc := range pvcList.Items {
			if pvc.Spec.VolumeName == "" {
				continue
			}

			pv, err := clientset.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
			if err != nil {
				return nil, nil, nil, err
			}

			storageClassName := ""
			if pvc.Spec.StorageClassName != nil {
				storageClassName = strings.TrimSpace(*pvc.Spec.StorageClassName)
			}
			provisioner := ""
			if storageClassName != "" {
				if storageClass, ok := storageClasses[storageClassName]; ok {
					provisioner = storageClass.Provisioner
				} else if storageClass, err := clientset.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{}); err == nil {
					storageClasses[storageClassName] = *storageClass
					provisioner = storageClass.Provisioner
				}
			}

			if pv.Spec.HostPath != nil {
				compatibilityErrors = append(compatibilityErrors, fmt.Sprintf("PVC %s/%s is backed by PV %s with hostPath, which Velero file system backup does not support", namespace, pvc.Name, pv.Name))
				unsupportedVolumes = append(unsupportedVolumes, veleromodel.UnsupportedVolume{
					Type:         "persistentVolume",
					Namespace:    namespace,
					PVC:          pvc.Name,
					PV:           pv.Name,
					StorageClass: storageClassName,
					Provisioner:  provisioner,
					Reason:       "hostPath-backed persistent volume is not supported by Velero file system backup",
				})
				continue
			}

			if provisioner == "rancher.io/local-path" {
				storageWarnings = append(storageWarnings, fmt.Sprintf("PVC %s/%s uses StorageClass %q (%s); verify the bound PV is not hostPath-backed before filesystem backup", namespace, pvc.Name, storageClassName, provisioner))
			}
		}
	}

	if len(unsupportedVolumes) > 0 {
		result.UnsupportedVolumes = unsupportedVolumes
	}
	if len(podWarnings) > 0 {
		result.PodWarnings = podWarnings
	}
	if len(storageWarnings) > 0 {
		result.StorageWarnings = storageWarnings
	}
	result.FilesystemBackupReady = len(compatibilityErrors) == 0 && len(podWarnings) == 0
	if len(compatibilityErrors) == 0 && len(podWarnings) == 0 {
		result.SupportedVolumeBackupModes = []string{veleromodel.VolumeBackupModeFilesystem}
		result.RecommendedVolumeBackupMode = veleromodel.VolumeBackupModeFilesystem
		result.RecommendedAction = "filesystem backup is acceptable for the inspected source volumes"
	} else {
		result.SupportedVolumeBackupModes = []string{}
		result.RecommendedVolumeBackupMode = ""
		result.RecommendedAction = "change the source storage class to a non-hostPath-backed volume or migrate PVC data outside Velero"
	}

	return result, storageWarnings, append(compatibilityErrors, podWarnings...), nil
}

func deriveBackupMode(defaultVolumesToFsBackup, snapshotVolumes *bool) string {
	if defaultVolumesToFsBackup != nil && *defaultVolumesToFsBackup {
		return veleromodel.VolumeBackupModeFilesystem
	}
	if snapshotVolumes != nil && *snapshotVolumes {
		return veleromodel.VolumeBackupModeSnapshot
	}
	return veleromodel.VolumeBackupModeFilesystem
}

func assessSnapshotCompatibility(
	ctx context.Context,
	sourceClientset, targetClientset *kubernetes.Clientset,
	sourceControllerClient, targetControllerClient ctrlclient.Client,
	namespaces []string,
	storageClassMappings map[string]string,
) (*veleromodel.SnapshotSupport, []string, []string, error) {
	result := &veleromodel.SnapshotSupport{}
	if len(namespaces) == 0 {
		result.SnapshotReady = false
		return result, nil, nil, nil
	}

	sourceDrivers, err := collectPVCProvisioners(ctx, sourceClientset, namespaces)
	if err != nil {
		return nil, nil, nil, err
	}
	targetDrivers, err := collectTargetMappedProvisioners(ctx, targetClientset, sourceDrivers, storageClassMappings)
	if err != nil {
		return nil, nil, nil, err
	}
	sourceSnapshotDrivers, err := collectVolumeSnapshotClassDrivers(ctx, sourceControllerClient)
	if err != nil {
		return nil, nil, nil, err
	}
	targetSnapshotDrivers, err := collectVolumeSnapshotClassDrivers(ctx, targetControllerClient)
	if err != nil {
		return nil, nil, nil, err
	}

	result.SourceProvisioners = sortedValues(sourceDrivers)
	result.TargetProvisioners = sortedValues(targetDrivers)
	result.SourceSnapshotDrivers = sortedKeys(sourceSnapshotDrivers)
	result.TargetSnapshotDrivers = sortedKeys(targetSnapshotDrivers)

	warnings := []string{}
	errors := []string{}
	snapshotReady := true

	if len(sourceSnapshotDrivers) == 0 {
		snapshotReady = false
		warnings = append(warnings, "source cluster does not expose VolumeSnapshotClass resources; snapshot backup is not available")
	}
	if len(targetSnapshotDrivers) == 0 {
		snapshotReady = false
		warnings = append(warnings, "target cluster does not expose VolumeSnapshotClass resources; snapshot restore is not available")
	}

	for sourceSC, sourceDriver := range sourceDrivers {
		targetDriver := targetDrivers[sourceSC]
		if sourceDriver == "" || targetDriver == "" {
			snapshotReady = false
			warnings = append(warnings, fmt.Sprintf("snapshot backup check skipped for storage class %q because source or target CSI provisioner could not be determined", sourceSC))
			continue
		}
		if _, ok := sourceSnapshotDrivers[sourceDriver]; !ok {
			snapshotReady = false
			warnings = append(warnings, fmt.Sprintf("source cluster does not have a VolumeSnapshotClass for CSI driver %q", sourceDriver))
		}
		if _, ok := targetSnapshotDrivers[targetDriver]; !ok {
			snapshotReady = false
			warnings = append(warnings, fmt.Sprintf("target cluster does not have a VolumeSnapshotClass for CSI driver %q", targetDriver))
		}
	}

	if len(sourceDrivers) == 0 {
		snapshotReady = false
	}
	result.SnapshotReady = snapshotReady
	return result, warnings, errors, nil
}

func collectPVCProvisioners(ctx context.Context, clientset *kubernetes.Clientset, namespaces []string) (map[string]string, error) {
	result := map[string]string{}
	for _, namespace := range namespaces {
		pvcList, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pvc := range pvcList.Items {
			if pvc.Spec.StorageClassName == nil || strings.TrimSpace(*pvc.Spec.StorageClassName) == "" {
				continue
			}
			scName := strings.TrimSpace(*pvc.Spec.StorageClassName)
			if _, exists := result[scName]; exists {
				continue
			}
			sc, err := clientset.StorageV1().StorageClasses().Get(ctx, scName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			result[scName] = sc.Provisioner
		}
	}
	return result, nil
}

func collectTargetMappedProvisioners(ctx context.Context, clientset *kubernetes.Clientset, sourceDrivers map[string]string, storageClassMappings map[string]string) (map[string]string, error) {
	result := map[string]string{}
	for sourceSC := range sourceDrivers {
		targetSC := sourceSC
		if mapped, ok := storageClassMappings[sourceSC]; ok && strings.TrimSpace(mapped) != "" {
			targetSC = strings.TrimSpace(mapped)
		}
		sc, err := clientset.StorageV1().StorageClasses().Get(ctx, targetSC, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		result[sourceSC] = sc.Provisioner
	}
	return result, nil
}

func collectVolumeSnapshotClassDrivers(ctx context.Context, controllerClient ctrlclient.Client) (map[string]struct{}, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "snapshot.storage.k8s.io",
		Version: "v1",
		Kind:    "VolumeSnapshotClassList",
	})

	if err := controllerClient.List(ctx, list); err != nil {
		if apierrors.IsNotFound(err) || metautils.IsNoMatchError(err) {
			return map[string]struct{}{}, nil
		}
		return nil, err
	}

	result := map[string]struct{}{}
	for _, item := range list.Items {
		driver, found, err := unstructured.NestedString(item.Object, "driver")
		if err != nil || !found || strings.TrimSpace(driver) == "" {
			driver, found, err = unstructured.NestedString(item.Object, "spec", "driver")
			if err != nil || !found || strings.TrimSpace(driver) == "" {
				continue
			}
		}
		result[strings.TrimSpace(driver)] = struct{}{}
	}
	return result, nil
}

func sortedKeys[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedValues(values map[string]string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsResource(values []string, target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	for _, value := range values {
		if strings.TrimSpace(strings.ToLower(value)) == target {
			return true
		}
	}
	return false
}

func buildSourceNamespaces(namespace string, includedNamespaces []string) []string {
	if namespace != "" {
		return []string{namespace}
	}
	return includedNamespaces
}

func normalizeRestoreSpec(spec veleromodel.RestoreSpec) (veleromodel.RestoreSpec, error) {
	if spec.Name == "" {
		spec.Name = fmt.Sprintf("restore-%d", time.Now().Unix())
	}
	if spec.SourceNamespace != "" && len(spec.IncludedNamespaces) == 0 {
		spec.IncludedNamespaces = []string{spec.SourceNamespace}
	}

	spec.NamespaceMapping = buildNamespaceMapping(spec.SourceNamespace, spec.TargetNamespace, spec.IncludedNamespaces, spec.NamespaceMapping)
	if spec.TargetNamespace != "" && len(spec.NamespaceMapping) == 0 {
		return spec, fmt.Errorf("targetNamespace requires namespace or a single included namespace")
	}
	if spec.RestorePVs == nil {
		restorePVs := true
		spec.RestorePVs = &restorePVs
	}

	switch spec.ExistingResourcePolicy {
	case "", string(velerov1.PolicyTypeNone), string(velerov1.PolicyTypeUpdate):
	default:
		return spec, fmt.Errorf("existingResourcePolicy must be one of: none, update")
	}

	return spec, nil
}

func uniqueMapValues(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func collectSourceStorageClasses(ctx context.Context, clientset *kubernetes.Clientset, namespaces []string) ([]string, error) {
	if len(namespaces) == 0 {
		return nil, nil
	}

	storageClasses := map[string]struct{}{}
	for _, namespace := range namespaces {
		pvcList, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pvc := range pvcList.Items {
			if pvc.Spec.StorageClassName == nil || strings.TrimSpace(*pvc.Spec.StorageClassName) == "" {
				continue
			}
			storageClasses[strings.TrimSpace(*pvc.Spec.StorageClassName)] = struct{}{}
		}
	}

	result := make([]string, 0, len(storageClasses))
	for name := range storageClasses {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func collectTargetStorageClasses(ctx context.Context, clientset *kubernetes.Clientset) (map[string]struct{}, error) {
	storageClassList, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := map[string]struct{}{}
	for _, storageClass := range storageClassList.Items {
		result[storageClass.Name] = struct{}{}
	}

	return result, nil
}

func buildNamespaceMapping(sourceNamespace, targetNamespace string, includedNamespaces []string, namespaceMapping map[string]string) map[string]string {
	if len(namespaceMapping) > 0 {
		return namespaceMapping
	}
	if targetNamespace == "" {
		return nil
	}
	if sourceNamespace != "" {
		return map[string]string{sourceNamespace: targetNamespace}
	}
	if len(includedNamespaces) == 1 {
		return map[string]string{includedNamespaces[0]: targetNamespace}
	}
	return nil
}

func (s *Service) ensureStorageClassConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace string, mappings map[string]string) error {
	if len(mappings) == 0 {
		return nil
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storageClassConfigMapName,
			Namespace: namespace,
			Labels: map[string]string{
				"velero.io/plugin-config":        "",
				"velero.io/change-storage-class": "RestoreItemAction",
				"app.kubernetes.io/managed-by":   "cm-grasshopper",
			},
		},
		Data: mappings,
	}

	existing, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, storageClassConfigMapName, metav1.GetOptions{})
	if err == nil {
		existing.Data = mappings
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		existing.Labels["velero.io/plugin-config"] = ""
		existing.Labels["velero.io/change-storage-class"] = "RestoreItemAction"
		existing.Labels["app.kubernetes.io/managed-by"] = "cm-grasshopper"
		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, existing, metav1.UpdateOptions{})
		return err
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}

func (s *Service) DeleteRestore(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) error {
	_, controllerClient, err := k8sclient.NewKubernetesClient(cluster)
	if err != nil {
		return err
	}

	namespace := k8scommon.DefaultNamespace(cluster, k8sinstaller.DefaultVeleroNamespace)
	return controllerClient.Delete(ctx, &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	})
}

func (s *Service) ValidateRestore(ctx context.Context, cluster *commonmodel.ClusterAccess, name string) (*veleromodel.RestoreResponse, error) {
	restore, err := s.GetRestore(ctx, cluster, name)
	if err != nil {
		return nil, err
	}
	return restore, nil
}

func (s *Service) waitForBackupCompletion(ctx context.Context, cluster *commonmodel.ClusterAccess, name, jobID string, progress int) (*velerov1.Backup, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, config.GetK8sBackupTimeout())
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timed out waiting for backup %s", name)
		case <-ticker.C:
			backup, err := s.getBackupResource(timeoutCtx, cluster, name)
			if err != nil {
				return nil, err
			}

			phase := strings.ToLower(string(backup.Status.Phase))
			updateJobProgressSafe(jobID, progress, fmt.Sprintf("Waiting for backup %s completion: %s", name, backup.Status.Phase))
			_ = joblib.DefaultManager.AddJobLog(jobID, fmt.Sprintf("Backup %s phase: %s", name, backup.Status.Phase))

			if phase == "completed" || phase == "partiallyfailed" {
				return backup, nil
			}
			if strings.Contains(phase, "failed") {
				return nil, fmt.Errorf("backup %s failed with phase %s", name, backup.Status.Phase)
			}
		}
	}
}

func (s *Service) waitForBackupSync(ctx context.Context, cluster *commonmodel.ClusterAccess, name, jobID string, progress int) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, config.GetK8sBackupTimeout())
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timed out waiting for backup %s to sync into target cluster", name)
		case <-ticker.C:
			backup, err := s.GetBackup(timeoutCtx, cluster, name)
			if err != nil {
				if apierrors.IsNotFound(err) {
					updateJobProgressSafe(jobID, progress, fmt.Sprintf("Waiting for backup %s to appear in target cluster", name))
					joblib.AddJobLogSafe(jobID, fmt.Sprintf("Waiting for backup %s to appear in target cluster", name))
					continue
				}
				return err
			}

			updateJobProgressSafe(jobID, progress, fmt.Sprintf("Backup %s synced into target cluster with phase %s", name, backup.Phase))
			joblib.AddJobLogSafe(jobID, fmt.Sprintf("Backup %s synced into target cluster with phase %s", name, backup.Phase))
			return nil
		}
	}
}

func (s *Service) waitForRestoreCompletion(ctx context.Context, cluster *commonmodel.ClusterAccess, name, jobID string, progress int) (*velerov1.Restore, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, config.GetK8sRestoreTimeout())
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("timed out waiting for restore %s", name)
		case <-ticker.C:
			restore, err := s.getRestoreResource(timeoutCtx, cluster, name)
			if err != nil {
				return nil, err
			}

			phase := strings.ToLower(string(restore.Status.Phase))
			updateJobProgressSafe(jobID, progress, fmt.Sprintf("Waiting for restore %s completion: %s", name, restore.Status.Phase))
			joblib.AddJobLogSafe(jobID, fmt.Sprintf("Restore %s phase: %s", name, restore.Status.Phase))

			if phase == "completed" || phase == "partiallyfailed" {
				return restore, nil
			}
			if len(restore.Status.ValidationErrors) > 0 || phase == "failedvalidation" {
				message := strings.Join(restore.Status.ValidationErrors, "; ")
				if strings.TrimSpace(message) == "" {
					message = "validation failed without a detailed error message"
				}
				return nil, fmt.Errorf("restore %s failed validation: %s", name, message)
			}
			if strings.Contains(phase, "failed") {
				return nil, fmt.Errorf("restore %s failed with phase %s", name, restore.Status.Phase)
			}
		}
	}
}
