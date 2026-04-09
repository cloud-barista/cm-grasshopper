package common

type ClusterAccess struct {
	Kubeconfig string `json:"kubeconfig"`
	Namespace  string `json:"namespace,omitempty"`
	Context    string `json:"context,omitempty"`
	Name       string `json:"name,omitempty"`
}

type MinIOAccess struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Bucket    string `json:"bucket,omitempty"`
	UseSSL    bool   `json:"useSSL"`
}

type StorageAccess struct {
	MinIO *MinIOAccess `json:"minio,omitempty"`
}

type MultiClusterEnvelope struct {
	SourceCluster *ClusterAccess `json:"sourceCluster,omitempty"`
	TargetCluster *ClusterAccess `json:"targetCluster,omitempty"`
	Storage       *StorageAccess `json:"storage,omitempty"`
}
