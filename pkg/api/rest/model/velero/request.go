package velero

import commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"

type InstallSpec struct {
	Force            bool   `json:"force"`
	VolumeBackupMode string `json:"volumeBackupMode,omitempty"`
}

type InstallRequest struct {
	commonmodel.MultiClusterEnvelope
	Install InstallSpec `json:"install"`
}

type BackupSpec struct {
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

type BackupRequest struct {
	commonmodel.MultiClusterEnvelope
	Backup BackupSpec `json:"backup"`
}

type RestoreSpec struct {
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
	RestorePVs              *bool             `json:"restorePVs,omitempty"`
}

type RestoreRequest struct {
	commonmodel.MultiClusterEnvelope
	Restore RestoreSpec `json:"restore"`
}

type MigrationPrecheckSpec struct {
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

type MigrationPrecheckRequest struct {
	commonmodel.MultiClusterEnvelope
	Precheck MigrationPrecheckSpec `json:"precheck"`
}

type MigrationExecuteSpec struct {
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
	RestorePVs               *bool             `json:"restorePVs,omitempty"`
}

type MigrationExecuteRequest struct {
	commonmodel.MultiClusterEnvelope
	Migration MigrationExecuteSpec `json:"migration"`
}
