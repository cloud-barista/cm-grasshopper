package common

import (
	"testing"

	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
)

// An unset prefix must land on "backups". A prefix explicitly set to "" keeps backups at
// the bucket root, which is where installs that never ran with force=true put them.
func TestDefaultS3Prefix(t *testing.T) {
	root := ""
	custom := "  team-a/velero  "

	cases := []struct {
		name string
		s3   *commonmodel.S3Access
		want string
	}{
		{"nil access falls back", nil, "backups"},
		{"unset prefix falls back", &commonmodel.S3Access{}, "backups"},
		{"explicit empty prefix means bucket root", &commonmodel.S3Access{Prefix: &root}, ""},
		{"explicit prefix is trimmed", &commonmodel.S3Access{Prefix: &custom}, "team-a/velero"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DefaultS3Prefix(tc.s3); got != tc.want {
				t.Errorf("DefaultS3Prefix() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDefaultS3Region(t *testing.T) {
	cases := []struct {
		name string
		s3   *commonmodel.S3Access
		want string
	}{
		{"nil access falls back", nil, "us-east-1"},
		{"unset region falls back", &commonmodel.S3Access{}, "us-east-1"},
		{"blank region falls back", &commonmodel.S3Access{Region: "   "}, "us-east-1"},
		{"explicit region is used", &commonmodel.S3Access{Region: "local"}, "local"},
		{"region is trimmed", &commonmodel.S3Access{Region: " ap-northeast-2 "}, "ap-northeast-2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DefaultS3Region(tc.s3); got != tc.want {
				t.Errorf("DefaultS3Region() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateClusterAccess(t *testing.T) {
	tb := &commonmodel.TumblebugK8sRef{NamespaceID: "mig01", K8sClusterID: "k8s-01"}
	hb := &commonmodel.HoneybeeK8sRef{SourceGroupID: "sg-01", ConnectionID: "conn-cp"}

	cases := []struct {
		name    string
		cluster *commonmodel.ClusterAccess
		wantErr string
	}{
		{"nil cluster", nil, "cluster access is required"},
		{"no source", &commonmodel.ClusterAccess{}, "one of kubeconfig, tumblebug or honeybee cluster reference is required"},
		{"blank kubeconfig only", &commonmodel.ClusterAccess{Kubeconfig: "   "}, "one of kubeconfig, tumblebug or honeybee cluster reference is required"},
		{"kubeconfig only", &commonmodel.ClusterAccess{Kubeconfig: "apiVersion: v1"}, ""},
		{"tumblebug only", &commonmodel.ClusterAccess{Tumblebug: tb}, ""},
		{"kubeconfig with tumblebug stays accepted", &commonmodel.ClusterAccess{Kubeconfig: "apiVersion: v1", Tumblebug: tb}, ""},
		{"tumblebug missing ids", &commonmodel.ClusterAccess{Tumblebug: &commonmodel.TumblebugK8sRef{NamespaceID: "mig01"}}, "tumblebug namespaceId and k8sClusterId are required"},
		{"honeybee only", &commonmodel.ClusterAccess{Honeybee: hb}, ""},
		{"honeybee without source group", &commonmodel.ClusterAccess{Honeybee: &commonmodel.HoneybeeK8sRef{ConnectionID: "conn-cp"}}, ""},
		{"honeybee with kubeconfig", &commonmodel.ClusterAccess{Kubeconfig: "apiVersion: v1", Honeybee: hb}, "honeybee cluster reference cannot be combined with kubeconfig or tumblebug"},
		{"honeybee with tumblebug", &commonmodel.ClusterAccess{Tumblebug: tb, Honeybee: hb}, "honeybee cluster reference cannot be combined with kubeconfig or tumblebug"},
		{"honeybee empty connectionId", &commonmodel.ClusterAccess{Honeybee: &commonmodel.HoneybeeK8sRef{SourceGroupID: "sg-01", ConnectionID: "  "}}, "honeybee connectionId is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateClusterAccess(tc.cluster)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateClusterAccess() error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("ValidateClusterAccess() error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

// A directly supplied kubeconfig wins over a tumblebug reference without any remote call.
func TestResolveKubeconfig_DirectKubeconfigFirst(t *testing.T) {
	raw := "apiVersion: v1\nkind: Config\n"
	cluster := &commonmodel.ClusterAccess{
		Kubeconfig: raw,
		Tumblebug:  &commonmodel.TumblebugK8sRef{NamespaceID: "mig01", K8sClusterID: "k8s-01"},
	}

	got, err := ResolveKubeconfig(cluster)
	if err != nil {
		t.Fatalf("ResolveKubeconfig() error = %v", err)
	}
	if got != raw {
		t.Fatalf("ResolveKubeconfig() = %q, want %q", got, raw)
	}
}
