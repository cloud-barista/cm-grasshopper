package software

import (
	"testing"

	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

// The software model of an AWS Ubuntu 22.04 source carries AWS guest agents and
// the lxd snap seeded by the Ubuntu cloud image. None of them may reach the
// migration list, while the applications next to them still must.
func TestProcessSoftwarePackagesDropsCSPAgentsAndImageSnaps(t *testing.T) {
	snap := softwaremodel.SoftwarePackageTypeSnap
	packages := []softwaremodel.Package{
		{Name: "amazon-ssm-agent", Type: snap, Channel: "latest/stable"},
		{Name: "lxd", Type: snap, Channel: "5.0/stable/ubuntu-22.04"},
		{Name: "ec2-hibinit-agent"},
		{Name: "ec2-instance-connect"},
		{Name: "hibagent"},
		{Name: "google-guest-agent"},
		{Name: "mariadb-server"},
		{Name: "nginx"},
		{Name: "certbot", Type: snap, Channel: "latest/stable"},
	}

	var order int
	got, _ := processSoftwarePackages(&order, packages)

	var names []string
	for _, p := range got {
		names = append(names, p.Name)
	}
	want := []string{"mariadb-server", "nginx", "certbot"}
	if len(names) != len(want) {
		t.Fatalf("migration list = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("migration list = %v, want %v", names, want)
		}
	}
	if got[2].Type != snap {
		t.Errorf("certbot type = %q, want %q", got[2].Type, snap)
	}
}

// A migration list built before the exclusion (e.g. frozen in a workflow) is
// filtered again at migrate time; those items must be recorded as skipped.
func TestBasePackageSkipReasonCSPAgents(t *testing.T) {
	for _, name := range []string{
		"amazon-ssm-agent", "ec2-hibinit-agent", "ec2-instance-connect", "hibagent",
		"google-guest-agent", "google-osconfig-agent", "lxd",
	} {
		if basePackageSkipReason(name) == "" {
			t.Errorf("%s: not skipped", name)
		}
	}
	for _, name := range []string{"mariadb-server", "nginx", "apache2", "nfs-kernel-server", "certbot"} {
		if r := basePackageSkipReason(name); r != "" {
			t.Errorf("%s: skipped (%s)", name, r)
		}
	}
}
