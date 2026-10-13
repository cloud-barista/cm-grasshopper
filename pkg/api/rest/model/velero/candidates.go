package velero

type CandidatesRequest struct {
	SourceGroupID string `json:"sourceGroupId"`
	ConnectionID  string `json:"connectionId"`
}

type CandidatePersistentVolumeClaim struct {
	Name         string   `json:"name"`
	StorageClass string   `json:"storageClass,omitempty"`
	AccessModes  []string `json:"accessModes,omitempty"`
}

type CandidateHelmRelease struct {
	Name         string `json:"name"`
	Chart        string `json:"chart,omitempty"`
	ChartVersion string `json:"chartVersion,omitempty"`
}

type CandidateNamespace struct {
	Name                   string                           `json:"name"`
	Workloads              map[string]int                   `json:"workloads"`
	PersistentVolumeClaims []CandidatePersistentVolumeClaim `json:"persistentVolumeClaims"`
	HelmReleases           []CandidateHelmRelease           `json:"helmReleases"`
}

type CandidateStorageClass struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner,omitempty"`
}

type CandidatesResponse struct {
	Version                string                  `json:"version"`
	Namespaces             []CandidateNamespace    `json:"namespaces"`
	ResourceKinds          []string                `json:"resourceKinds"`
	StorageClasses         []CandidateStorageClass `json:"storageClasses"`
	ClusterScopedWorkloads map[string]int          `json:"clusterScopedWorkloads"`
}
