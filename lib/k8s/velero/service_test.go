package velero

import (
	"context"
	"strings"
	"testing"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func storageClass(name, provisioner string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta:  metav1.ObjectMeta{Name: name},
		Provisioner: provisioner,
	}
}

func pvcWithStorageClass(namespace, name, storageClassName string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec:       v1.PersistentVolumeClaimSpec{StorageClassName: &storageClassName},
	}
}

// A source StorageClass that does not exist on the target is the normal case for a
// cross-CSP migration - AWS gp2 is never present on AKS. Precheck exists to report that as
// a missing mapping, so the lookup must not turn it into a failure.
func TestCollectTargetMappedProvisioners_MissingTargetStorageClassIsSkipped(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	sourceDrivers := map[string]string{"gp2": "kubernetes.io/aws-ebs"}

	result, err := collectTargetMappedProvisioners(context.Background(), clientset, sourceDrivers, nil)
	if err != nil {
		t.Fatalf("missing target StorageClass must not fail the precheck: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected no provisioners resolved, got %v", result)
	}
}

func TestCollectTargetMappedProvisioners_MappedStorageClassResolves(t *testing.T) {
	clientset := fake.NewSimpleClientset(storageClass("managed-csi", "disk.csi.azure.com"))
	sourceDrivers := map[string]string{"gp2": "kubernetes.io/aws-ebs"}

	result, err := collectTargetMappedProvisioners(context.Background(), clientset, sourceDrivers,
		map[string]string{"gp2": "managed-csi"})
	if err != nil {
		t.Fatalf("collectTargetMappedProvisioners: %v", err)
	}
	if got := result["gp2"]; got != "disk.csi.azure.com" {
		t.Errorf("result[gp2] = %q, want %q", got, "disk.csi.azure.com")
	}
}

// A PVC can outlive the StorageClass it was created from. That is a reportable condition,
// not a reason to abort the whole precheck.
func TestCollectPVCProvisioners_MissingStorageClassIsSkipped(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		pvcWithStorageClass("demo", "orphan", "deleted-sc"),
		pvcWithStorageClass("demo", "live", "gp2"),
		storageClass("gp2", "kubernetes.io/aws-ebs"),
	)

	result, err := collectPVCProvisioners(context.Background(), clientset, []string{"demo"})
	if err != nil {
		t.Fatalf("missing StorageClass must not fail the precheck: %v", err)
	}
	if got := result["gp2"]; got != "kubernetes.io/aws-ebs" {
		t.Errorf("result[gp2] = %q, want %q", got, "kubernetes.io/aws-ebs")
	}
	if _, found := result["deleted-sc"]; found {
		t.Errorf("a StorageClass that no longer exists must not appear in the result: %v", result)
	}
}

func bsl(bucket, prefix string) *velerov1.BackupStorageLocation {
	return &velerov1.BackupStorageLocation{
		Spec: velerov1.BackupStorageLocationSpec{
			StorageType: velerov1.StorageType{
				ObjectStorage: &velerov1.ObjectStorageLocation{Bucket: bucket, Prefix: prefix},
			},
		},
	}
}

// A source writing under "backups/" while the target reads the bucket root produced no
// error at all - the migration sat at "waiting for backup to appear" until the backup
// timeout expired, and the timeout message named neither bucket nor prefix.
func TestBackupLocationMismatch(t *testing.T) {
	cases := []struct {
		name           string
		source, target *velerov1.BackupStorageLocation
		wantErr        bool
	}{
		{"identical locations match", bsl("velero", "backups"), bsl("velero", "backups"), false},
		{"both at bucket root match", bsl("velero", ""), bsl("velero", ""), false},
		{"prefix mismatch is reported", bsl("velero", "backups"), bsl("velero", ""), true},
		{"bucket mismatch is reported", bsl("velero", "backups"), bsl("other", "backups"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := backupLocationMismatch(tc.source, tc.target)
			if tc.wantErr == (err == nil) {
				t.Fatalf("backupLocationMismatch() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// The whole point of failing fast is telling the operator which side to fix.
func TestBackupLocationMismatch_ErrorNamesBothSides(t *testing.T) {
	err := backupLocationMismatch(bsl("velero", "backups"), bsl("velero", ""))
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{"velero/backups", "velero/", "source", "target"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message missing %q: %s", want, err)
		}
	}
}

func TestBackupLocationMismatch_MissingObjectStorage(t *testing.T) {
	empty := &velerov1.BackupStorageLocation{}

	if err := backupLocationMismatch(empty, bsl("velero", "backups")); err == nil {
		t.Error("a location without objectStorage must be reported, not silently accepted")
	}
}
