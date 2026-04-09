package velero

const (
	VolumeBackupModeFilesystem = "filesystem"
	VolumeBackupModeSnapshot   = "snapshot"
	NameConflictPolicyRename   = "rename"
	NameConflictPolicyFail     = "fail"
)

type StorageClassRecommendation struct {
	MappingRequired   bool              `json:"mappingRequired"`
	UsedMappings      map[string]string `json:"usedMappings,omitempty"`
	UnusedMappings    map[string]string `json:"unusedMappings,omitempty"`
	MissingMappings   []string          `json:"missingMappings,omitempty"`
	SuggestedMappings map[string]string `json:"suggestedMappings,omitempty"`
}

type UnsupportedVolume struct {
	Type         string `json:"type"`
	Namespace    string `json:"namespace,omitempty"`
	Pod          string `json:"pod,omitempty"`
	Volume       string `json:"volume,omitempty"`
	PVC          string `json:"pvc,omitempty"`
	PV           string `json:"pv,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
	Provisioner  string `json:"provisioner,omitempty"`
	Reason       string `json:"reason"`
}

type SnapshotSupport struct {
	SnapshotReady         bool     `json:"snapshotReady"`
	SourceProvisioners    []string `json:"sourceProvisioners,omitempty"`
	TargetProvisioners    []string `json:"targetProvisioners,omitempty"`
	SourceSnapshotDrivers []string `json:"sourceSnapshotDrivers,omitempty"`
	TargetSnapshotDrivers []string `json:"targetSnapshotDrivers,omitempty"`
}

type VolumeBackupCompatibility struct {
	VolumeBackupMode            string              `json:"volumeBackupMode,omitempty"`
	FilesystemBackupReady       bool                `json:"filesystemBackupReady"`
	SupportedVolumeBackupModes  []string            `json:"supportedVolumeBackupModes,omitempty"`
	RecommendedVolumeBackupMode string              `json:"recommendedVolumeBackupMode,omitempty"`
	RecommendedAction           string              `json:"recommendedAction,omitempty"`
	UnsupportedVolumes          []UnsupportedVolume `json:"unsupportedVolumes,omitempty"`
	PodWarnings                 []string            `json:"podWarnings,omitempty"`
	StorageWarnings             []string            `json:"storageWarnings,omitempty"`
	SnapshotSupport             *SnapshotSupport    `json:"snapshotSupport,omitempty"`
}
