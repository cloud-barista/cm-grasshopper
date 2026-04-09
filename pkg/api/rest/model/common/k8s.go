package common

type ClusterAccess struct {
	Kubeconfig string `json:"kubeconfig"`
	Namespace  string `json:"namespace,omitempty"`
	Context    string `json:"context,omitempty"`
	Name       string `json:"name,omitempty"`
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
