package software

import (
	"testing"

	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

func TestResolveNodeSourceMachineID(t *testing.T) {
	group := map[string]string{sourceMachineIdsLabelKey: "srv-01, srv-02,srv-03"}

	tests := []struct {
		name string
		node model.TBNodeInfo
		want string
	}{
		{"single id", model.TBNodeInfo{Name: "g1-1", Label: map[string]string{sourceMachineIdsLabelKey: "srv-01"}}, "srv-01"},
		{"single id without index suffix", model.TBNodeInfo{Name: "g1", Label: map[string]string{sourceMachineIdsLabelKey: "srv-01"}}, "srv-01"},
		{"group first", model.TBNodeInfo{Name: "web-g-1", Label: group}, "srv-01"},
		{"group second", model.TBNodeInfo{Name: "web-g-2", Label: group}, "srv-02"},
		{"group third", model.TBNodeInfo{Name: "web-g-3", Label: group}, "srv-03"},
		{"group index out of range", model.TBNodeInfo{Name: "web-g-4", Label: group}, ""},
		{"group without index suffix", model.TBNodeInfo{Name: "web-g", Label: group}, ""},
		{"no label", model.TBNodeInfo{Name: "g1-1"}, ""},
		{"legacy singular key", model.TBNodeInfo{Name: "g1-1", Label: map[string]string{"sourceMachineId": "srv-01"}}, ""},
		{"legacy plural key", model.TBNodeInfo{Name: "g1-1", Label: map[string]string{"sourceMachineIds": "srv-01"}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveNodeSourceMachineID(tt.node); got != tt.want {
				t.Errorf("resolveNodeSourceMachineID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKubernetesSkippedStatus(t *testing.T) {
	k := softwaremodel.KubernetesMigrationInfo{Order: 7, Version: "1.30.2"}

	got := kubernetesSkippedStatus("ex-1", "conn-1", "ns-1", "infra-1", "node-1", k)

	want := model.SoftwareMigrationStatus{
		ExecutionID:            "ex-1",
		SourceConnectionInfoID: "conn-1",
		NamespaceID:            "ns-1",
		InfraID:                "infra-1",
		NodeID:                 "node-1",
		Order:                  7,
		SoftwareName:           "kubernetes",
		SoftwareVersion:        "1.30.2",
		SoftwareInstallType:    softwaremodel.SoftwareTypeKubernetes,
		Status:                 "skipped",
		ErrorMessage:           "Kubernetes migration is served by POST /grasshopper/k8s/migrate",
	}
	if got != want {
		t.Errorf("kubernetesSkippedStatus() = %+v, want %+v", got, want)
	}
}
