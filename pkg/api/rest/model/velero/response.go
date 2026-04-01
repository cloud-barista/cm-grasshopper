package velero

type BackupStorageLocationHealth struct {
	Name               string `json:"name"`
	Phase              string `json:"phase"`
	Message            string `json:"message,omitempty"`
	LastValidationTime string `json:"lastValidationTime,omitempty"`
}

type HealthResponse struct {
	Status                string                       `json:"status"`
	Cluster               string                       `json:"cluster,omitempty"`
	Namespace             string                       `json:"namespace"`
	BackupStorageLocation *BackupStorageLocationHealth `json:"backupStorageLocation,omitempty"`
}

type PrecheckStorageSummary struct {
	Endpoint string `json:"endpoint"`
	Bucket   string `json:"bucket"`
}

type PrecheckClusterSummary struct {
	Name                       string                       `json:"name"`
	Namespace                  string                       `json:"namespace"`
	BackupStorageLocation      *BackupStorageLocationHealth `json:"backupStorageLocation,omitempty"`
	SourceNamespaces           []string                     `json:"sourceNamespaces,omitempty"`
	NamespaceStatus            map[string]string            `json:"namespaceStatus,omitempty"`
	StorageClasses             []string                     `json:"storageClasses,omitempty"`
	StorageClassRecommendation *StorageClassRecommendation  `json:"storageClassRecommendation,omitempty"`
	VolumeBackupCompatibility  *VolumeBackupCompatibility   `json:"volumeBackupCompatibility,omitempty"`
}

type PrecheckResponse struct {
	Status   string                 `json:"status"`
	Source   PrecheckClusterSummary `json:"source"`
	Target   PrecheckClusterSummary `json:"target"`
	Storage  PrecheckStorageSummary `json:"storage"`
	Warnings []string               `json:"warnings"`
	Errors   []string               `json:"errors"`
}

type PrecheckSummary struct {
	RecommendedVolumeBackupMode string            `json:"recommendedVolumeBackupMode,omitempty"`
	SourceNamespaces            []string          `json:"sourceNamespaces,omitempty"`
	TargetNamespaces            map[string]string `json:"targetNamespaces,omitempty"`
	StorageClassMappingRequired bool              `json:"storageClassMappingRequired"`
	BackupStorageLocationReady  bool              `json:"backupStorageLocationReady"`
}

type PrecheckCompactResponse struct {
	Status   string          `json:"status"`
	Summary  PrecheckSummary `json:"summary"`
	Warnings []string        `json:"warnings"`
	Errors   []string        `json:"errors"`
}

type BackupResponse struct {
	Name                      string                     `json:"name"`
	RequestedName             string                     `json:"requestedName,omitempty"`
	NameAdjusted              bool                       `json:"nameAdjusted,omitempty"`
	Namespace                 string                     `json:"namespace"`
	Phase                     string                     `json:"phase"`
	Warnings                  int                        `json:"warnings"`
	Errors                    int                        `json:"errors"`
	CreatedAt                 string                     `json:"createdAt,omitempty"`
	Started                   string                     `json:"started,omitempty"`
	Completed                 string                     `json:"completed,omitempty"`
	BackupMode                string                     `json:"backupMode"`
	Storage                   string                     `json:"storage,omitempty"`
	TTL                       string                     `json:"ttl,omitempty"`
	IncludedNamespaces        []string                   `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces        []string                   `json:"excludedNamespaces,omitempty"`
	IncludedResources         []string                   `json:"includedResources,omitempty"`
	ExcludedResources         []string                   `json:"excludedResources,omitempty"`
	IncludeClusterResources   *bool                      `json:"includeClusterResources,omitempty"`
	DefaultVolumesToFsBackup  *bool                      `json:"defaultVolumesToFsBackup,omitempty"`
	SnapshotVolumes           *bool                      `json:"snapshotVolumes,omitempty"`
	VolumeBackupCompatibility *VolumeBackupCompatibility `json:"volumeBackupCompatibility,omitempty"`
	CompatibilityWarnings     []string                   `json:"compatibilityWarnings,omitempty"`
	CompatibilityErrors       []string                   `json:"compatibilityErrors,omitempty"`
}

type RestoreResponse struct {
	Name                    string            `json:"name"`
	Namespace               string            `json:"namespace"`
	Phase                   string            `json:"phase"`
	Warnings                int               `json:"warnings"`
	Errors                  int               `json:"errors"`
	ValidationErrors        []string          `json:"validationErrors,omitempty"`
	CreatedAt               string            `json:"createdAt,omitempty"`
	Started                 string            `json:"started,omitempty"`
	Completed               string            `json:"completed,omitempty"`
	BackupName              string            `json:"backupName"`
	RestorePVs              *bool             `json:"restorePVs,omitempty"`
	ExistingResourcePolicy  string            `json:"existingResourcePolicy,omitempty"`
	IncludedNamespaces      []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string          `json:"includedResources,omitempty"`
	ExcludedResources       []string          `json:"excludedResources,omitempty"`
	NamespaceMapping        map[string]string `json:"namespaceMapping,omitempty"`
	IncludeClusterResources *bool             `json:"includeClusterResources,omitempty"`
}
