package model

const (
	VeleroVolumeBackupModeFilesystem = "filesystem"
	VeleroVolumeBackupModeSnapshot   = "snapshot"
	VeleroNameConflictPolicyRename   = "rename"
	VeleroNameConflictPolicyFail     = "fail"
)

type VeleroInstallSpec struct {
	Force            bool   `json:"force"`
	VolumeBackupMode string `json:"volumeBackupMode,omitempty"`
}

type VeleroInstallRequest struct {
	MultiClusterEnvelope
	Install VeleroInstallSpec `json:"install"`
}

type VeleroBackupSpec struct {
	Name                     string   `json:"name"`
	SourceNamespace          string   `json:"sourceNamespace,omitempty"`
	IncludedNamespaces       []string `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces       []string `json:"excludedNamespaces,omitempty"`
	IncludedResources        []string `json:"includedResources,omitempty"`
	ExcludedResources        []string `json:"excludedResources,omitempty"`
	IncludeClusterResources  *bool    `json:"includeClusterResources,omitempty"`
	VolumeBackupMode         string   `json:"volumeBackupMode,omitempty"`
	NameConflictPolicy       string   `json:"nameConflictPolicy,omitempty"`
	SnapshotVolumes          bool     `json:"snapshotVolumes"`
	DefaultVolumesToFsBackup bool     `json:"defaultVolumesToFsBackup"`
}

type VeleroBackupRequest struct {
	MultiClusterEnvelope
	Backup VeleroBackupSpec `json:"backup"`
}

type VeleroRestoreSpec struct {
	Name                    string            `json:"name"`
	BackupName              string            `json:"backupName"`
	SourceNamespace         string            `json:"sourceNamespace,omitempty"`
	TargetNamespace         string            `json:"targetNamespace,omitempty"`
	IncludedNamespaces      []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string          `json:"includedResources,omitempty"`
	ExcludedResources       []string          `json:"excludedResources,omitempty"`
	NamespaceMapping        map[string]string `json:"namespaceMapping,omitempty"`
	StorageClassMappings    map[string]string `json:"storageClassMappings,omitempty"`
	IncludeClusterResources *bool             `json:"includeClusterResources,omitempty"`
	ExistingResourcePolicy  string            `json:"existingResourcePolicy,omitempty"`
	RestorePVs              bool              `json:"restorePVs"`
}

type VeleroRestoreRequest struct {
	MultiClusterEnvelope
	Restore VeleroRestoreSpec `json:"restore"`
}

type VeleroMigrationPrecheckSpec struct {
	BackupName              string            `json:"backupName,omitempty"`
	RestoreName             string            `json:"restoreName,omitempty"`
	SourceNamespace         string            `json:"sourceNamespace,omitempty"`
	TargetNamespace         string            `json:"targetNamespace,omitempty"`
	IncludedNamespaces      []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string          `json:"includedResources,omitempty"`
	ExcludedResources       []string          `json:"excludedResources,omitempty"`
	NamespaceMapping        map[string]string `json:"namespaceMapping,omitempty"`
	StorageClassMappings    map[string]string `json:"storageClassMappings,omitempty"`
	IncludeClusterResources *bool             `json:"includeClusterResources,omitempty"`
	VolumeBackupMode        string            `json:"volumeBackupMode,omitempty"`
}

type VeleroMigrationPrecheckRequest struct {
	MultiClusterEnvelope
	Precheck VeleroMigrationPrecheckSpec `json:"precheck"`
}

type VeleroMigrationExecuteSpec struct {
	BackupName               string            `json:"backupName,omitempty"`
	RestoreName              string            `json:"restoreName,omitempty"`
	SourceNamespace          string            `json:"sourceNamespace,omitempty"`
	TargetNamespace          string            `json:"targetNamespace,omitempty"`
	IncludedNamespaces       []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces       []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources        []string          `json:"includedResources,omitempty"`
	ExcludedResources        []string          `json:"excludedResources,omitempty"`
	NamespaceMapping         map[string]string `json:"namespaceMapping,omitempty"`
	StorageClassMappings     map[string]string `json:"storageClassMappings,omitempty"`
	IncludeClusterResources  *bool             `json:"includeClusterResources,omitempty"`
	ExistingResourcePolicy   string            `json:"existingResourcePolicy,omitempty"`
	VolumeBackupMode         string            `json:"volumeBackupMode,omitempty"`
	NameConflictPolicy       string            `json:"nameConflictPolicy,omitempty"`
	SnapshotVolumes          bool              `json:"snapshotVolumes"`
	DefaultVolumesToFsBackup bool              `json:"defaultVolumesToFsBackup"`
	RestorePVs               bool              `json:"restorePVs"`
}

type VeleroMigrationExecuteRequest struct {
	MultiClusterEnvelope
	Migration VeleroMigrationExecuteSpec `json:"migration"`
}

type VeleroBackupStorageLocationHealth struct {
	Name               string      `json:"name"`
	Phase              string      `json:"phase"`
	Message            string      `json:"message,omitempty"`
	LastValidationTime interface{} `json:"lastValidationTime,omitempty"`
}

type VeleroHealthResponse struct {
	Status                string                            `json:"status"`
	Cluster               string                            `json:"cluster,omitempty"`
	Namespace             string                            `json:"namespace"`
	BackupStorageLocation *VeleroBackupStorageLocationHealth `json:"backupStorageLocation,omitempty"`
}

type VeleroPrecheckStorageSummary struct {
	Endpoint string `json:"endpoint"`
	Bucket   string `json:"bucket"`
}

type VeleroPrecheckClusterSummary struct {
	Name                      string                             `json:"name"`
	Namespace                 string                             `json:"namespace"`
	BackupStorageLocation     *VeleroBackupStorageLocationHealth `json:"backupStorageLocation,omitempty"`
	SourceNamespaces          []string                           `json:"sourceNamespaces,omitempty"`
	NamespaceStatus           map[string]string                  `json:"namespaceStatus,omitempty"`
	StorageClasses            interface{}                        `json:"storageClasses,omitempty"`
	StorageClassRecommendation map[string]interface{}            `json:"storageClassRecommendation,omitempty"`
	VolumeBackupCompatibility map[string]interface{}             `json:"volumeBackupCompatibility,omitempty"`
}

type VeleroPrecheckResponse struct {
	Status   string                       `json:"status"`
	Source   VeleroPrecheckClusterSummary `json:"source"`
	Target   VeleroPrecheckClusterSummary `json:"target"`
	Storage  VeleroPrecheckStorageSummary `json:"storage"`
	Warnings []string                     `json:"warnings"`
	Errors   []string                     `json:"errors"`
}

type VeleroPrecheckSummary struct {
	RecommendedVolumeBackupMode string            `json:"recommendedVolumeBackupMode,omitempty"`
	SourceNamespaces            []string          `json:"sourceNamespaces,omitempty"`
	TargetNamespaces            map[string]string `json:"targetNamespaces,omitempty"`
	StorageClassMappingRequired bool              `json:"storageClassMappingRequired"`
	BackupStorageLocationReady  bool              `json:"backupStorageLocationReady"`
}

type VeleroPrecheckCompactResponse struct {
	Status   string                `json:"status"`
	Summary  VeleroPrecheckSummary `json:"summary"`
	Warnings []string              `json:"warnings"`
	Errors   []string              `json:"errors"`
}

type VeleroBackupResponse struct {
	Name                     string      `json:"name"`
	RequestedName            string      `json:"requestedName,omitempty"`
	NameAdjusted             bool        `json:"nameAdjusted,omitempty"`
	Namespace                string      `json:"namespace"`
	Phase                    string      `json:"phase"`
	Warnings                 int         `json:"warnings"`
	Errors                   int         `json:"errors"`
	CreatedAt                interface{} `json:"createdAt,omitempty"`
	Started                  interface{} `json:"started,omitempty"`
	Completed                interface{} `json:"completed,omitempty"`
	BackupMode               string      `json:"backupMode"`
	Storage                  string      `json:"storage,omitempty"`
	TTL                      string      `json:"ttl,omitempty"`
	IncludedNamespaces       []string    `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces       []string    `json:"excludedNamespaces,omitempty"`
	IncludedResources        []string    `json:"includedResources,omitempty"`
	ExcludedResources        []string    `json:"excludedResources,omitempty"`
	IncludeClusterResources  *bool       `json:"includeClusterResources,omitempty"`
	DefaultVolumesToFsBackup *bool       `json:"defaultVolumesToFsBackup,omitempty"`
	SnapshotVolumes          *bool       `json:"snapshotVolumes,omitempty"`
	VolumeBackupCompatibility interface{} `json:"volumeBackupCompatibility,omitempty"`
	CompatibilityWarnings    []string    `json:"compatibilityWarnings,omitempty"`
	CompatibilityErrors      []string    `json:"compatibilityErrors,omitempty"`
}
