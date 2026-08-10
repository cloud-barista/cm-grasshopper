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
