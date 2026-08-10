package installer

import (
	"strings"
	"testing"

	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
	veleromodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/velero"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	v1 "k8s.io/api/core/v1"
)

func testS3(region string) *commonmodel.S3Access {
	return &commonmodel.S3Access{
		Endpoint:  "minio-api.example.com",
		AccessKey: "AK",
		SecretKey: "SK",
		Bucket:    "velero",
		Region:    region,
		UseSSL:    true,
	}
}

func TestBuildCloudCredentials_Region(t *testing.T) {
	got := buildCloudCredentials(testS3("local"))

	for _, want := range []string{
		"[default]",
		"aws_access_key_id=AK",
		"aws_secret_access_key=SK",
		"region=local",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("credentials missing %q\n got:\n%s", want, got)
		}
	}
}

func TestBuildCloudCredentials_RegionFallback(t *testing.T) {
	got := buildCloudCredentials(testS3(""))

	if !strings.Contains(got, "region=us-east-1") {
		t.Errorf("unset region must fall back to %q\n got:\n%s", "us-east-1", got)
	}
}

func TestBuildVeleroValues_Region(t *testing.T) {
	values := buildVeleroValues(testS3("local"), veleromodel.VolumeBackupModeFilesystem)

	config := backupStorageLocationConfig(t, values)
	if got := config["region"]; got != "local" {
		t.Errorf("backupStorageLocation config region = %q, want %q", got, "local")
	}
}

func TestBuildVeleroValues_RegionDefaults(t *testing.T) {
	values := buildVeleroValues(testS3(""), veleromodel.VolumeBackupModeFilesystem)

	config := backupStorageLocationConfig(t, values)
	if got := config["region"]; got != "us-east-1" {
		t.Errorf("backupStorageLocation config region = %q, want %q", got, "us-east-1")
	}
}

// The Helm chart and ensureBackupStorageLocation both write the BSL. When only the latter
// set a prefix, whether backups landed under "backups/" or at the bucket root depended on
// the install's force flag, and a source/target pair installed differently never synced.
func TestBuildBSLSpec_MatchesHelmValues(t *testing.T) {
	root := ""

	for _, tc := range []struct {
		name string
		s3   *commonmodel.S3Access
	}{
		{"default prefix", testS3("local")},
		{"bucket root", withPrefix(testS3("local"), &root)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := buildBSLSpec(tc.s3)
			if err != nil {
				t.Fatalf("buildBSLSpec: %v", err)
			}
			location := helmBackupStorageLocation(t, buildVeleroValues(tc.s3, veleromodel.VolumeBackupModeFilesystem))

			if got, want := spec.ObjectStorage.Bucket, location["bucket"]; got != want {
				t.Errorf("bucket: BSL spec = %q, helm values = %v", got, want)
			}
			if got, want := spec.ObjectStorage.Prefix, location["prefix"]; got != want {
				t.Errorf("prefix: BSL spec = %q, helm values = %v", got, want)
			}
		})
	}
}

func TestBuildBSLSpec_DefaultPrefix(t *testing.T) {
	spec, err := buildBSLSpec(testS3("local"))
	if err != nil {
		t.Fatalf("buildBSLSpec: %v", err)
	}

	if spec.ObjectStorage.Prefix != "backups" {
		t.Errorf("prefix = %q, want %q", spec.ObjectStorage.Prefix, "backups")
	}
}

// ensureBackupStorageLocation used to return early whenever a BSL already existed unless
// force was set, so a reinstall pointing at a different endpoint or prefix silently kept
// the old one.
func TestBSLNeedsUpdate(t *testing.T) {
	root := ""
	base, err := buildBSLSpec(testS3("local"))
	if err != nil {
		t.Fatalf("buildBSLSpec: %v", err)
	}

	rootPrefix, _ := buildBSLSpec(withPrefix(testS3("local"), &root))
	otherRegion, _ := buildBSLSpec(testS3("ap-northeast-2"))

	cases := []struct {
		name    string
		desired velerov1.BackupStorageLocationSpec
		want    bool
	}{
		{"identical spec needs no update", base, false},
		{"prefix change needs update", rootPrefix, true},
		{"region change needs update", otherRegion, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			existing := &velerov1.BackupStorageLocation{Spec: base}
			if got := bslNeedsUpdate(existing, tc.desired); got != tc.want {
				t.Errorf("bslNeedsUpdate() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ensureSecret returned early whenever cloud-credentials existed unless force was set, so a
// reinstall with rotated keys kept the old ones while the Helm-managed BSL moved to the new
// endpoint - an auth failure whose cause is nowhere near the change that produced it.
func TestCloudCredentialsChanged(t *testing.T) {
	current := buildCloudCredentials(testS3("local"))

	cases := []struct {
		name     string
		existing *v1.Secret
		want     bool
	}{
		{"same content needs no update", secretWithCloud(current), false},
		{"rotated keys need update", secretWithCloud(buildCloudCredentials(&commonmodel.S3Access{
			Endpoint: "minio-api.example.com", AccessKey: "NEW", SecretKey: "NEW", Region: "local",
		})), true},
		{"region change needs update", secretWithCloud(buildCloudCredentials(testS3("ap-northeast-2"))), true},
		{"missing key needs update", &v1.Secret{}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := cloudCredentialsChanged(tc.existing, current); got != tc.want {
				t.Errorf("cloudCredentialsChanged() = %v, want %v", got, tc.want)
			}
		})
	}
}

func secretWithCloud(content string) *v1.Secret {
	return &v1.Secret{Data: map[string][]byte{"cloud": []byte(content)}}
}

func withPrefix(s3 *commonmodel.S3Access, prefix *string) *commonmodel.S3Access {
	clone := *s3
	clone.Prefix = prefix

	return &clone
}

// Divergence between the two config sources made an install report success while the
// BackupStorageLocation sat Unavailable.
func TestBuildBSLConfig_MatchesHelmValues(t *testing.T) {
	for _, region := range []string{"local", ""} {
		s3 := testS3(region)

		bslConfig, err := buildBSLConfig(s3)
		if err != nil {
			t.Fatalf("buildBSLConfig(region=%q): %v", region, err)
		}
		helmConfig := backupStorageLocationConfig(t, buildVeleroValues(s3, veleromodel.VolumeBackupModeFilesystem))

		for _, key := range []string{"region", "s3Url", "s3ForcePathStyle"} {
			want, ok := helmConfig[key].(string)
			if !ok {
				t.Fatalf("helm values config[%q] is not a string: %#v", key, helmConfig[key])
			}
			if bslConfig[key] != want {
				t.Errorf("region=%q: BSL config[%q] = %q, helm values = %q", region, key, bslConfig[key], want)
			}
		}
	}
}

func helmBackupStorageLocation(t *testing.T, values map[string]interface{}) map[string]interface{} {
	t.Helper()

	configuration, ok := values["configuration"].(map[string]interface{})
	if !ok {
		t.Fatalf("values.configuration is not a map: %#v", values["configuration"])
	}
	locations, ok := configuration["backupStorageLocation"].([]interface{})
	if !ok || len(locations) != 1 {
		t.Fatalf("expected exactly one backupStorageLocation, got %#v", configuration["backupStorageLocation"])
	}
	location, ok := locations[0].(map[string]interface{})
	if !ok {
		t.Fatalf("backupStorageLocation[0] is not a map: %#v", locations[0])
	}

	return location
}

func backupStorageLocationConfig(t *testing.T, values map[string]interface{}) map[string]interface{} {
	t.Helper()

	location := helmBackupStorageLocation(t, values)
	config, ok := location["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("backupStorageLocation[0].config is not a map: %#v", location["config"])
	}

	return config
}
