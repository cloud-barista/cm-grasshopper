package common

type ClusterAccess struct {
	Kubeconfig string `json:"kubeconfig"`
	Namespace  string `json:"namespace,omitempty"`
	Context    string `json:"context,omitempty"`
	Name       string `json:"name,omitempty"`

	// Tumblebug optionally references a cb-tumblebug-managed k8s cluster. When set and
	// Kubeconfig is empty, the kubeconfig is resolved from cb-tumblebug at request time.
	Tumblebug *TumblebugK8sRef `json:"tumblebug,omitempty"`
}

// TumblebugK8sRef identifies a Kubernetes cluster deployed through cb-tumblebug so that
// cm-grasshopper can resolve its kubeconfig (and, for exec-plugin CSPs such as AWS EKS /
// GCP GKE / NCP NKS, a self-contained token-broker kubeconfig) without any cloud CLI.
type TumblebugK8sRef struct {
	NamespaceID  string `json:"namespaceId"`
	K8sClusterID string `json:"k8sClusterId"`
}

// S3Access describes credentials for any S3-compatible object store
// (RustFS, MinIO, Garage, Ceph RGW, AWS S3, ...).
type S3Access struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Bucket    string `json:"bucket,omitempty"`
	UseSSL    bool   `json:"useSSL"`
}

type StorageAccess struct {
	S3 *S3Access `json:"s3,omitempty"`
}

type MultiClusterEnvelope struct {
	SourceCluster *ClusterAccess `json:"sourceCluster,omitempty"`
	TargetCluster *ClusterAccess `json:"targetCluster,omitempty"`
	Storage       *StorageAccess `json:"storage,omitempty"`
}
