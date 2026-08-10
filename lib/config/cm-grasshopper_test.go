package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// Existing deployments have no features block. Their behaviour must not change on upgrade,
// so an absent flag means enabled.
func TestFeatureFlagsDefaultToEnabled(t *testing.T) {
	restore := swapConfig(t)
	defer restore()

	if err := yaml.Unmarshal([]byte("cm-grasshopper:\n    listen:\n        port: \"8084\"\n"), &CMGrasshopperConfig); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !SoftwareMigrationEnabled() {
		t.Error("software migration must default to enabled")
	}
	if !K8sMigrationEnabled() {
		t.Error("k8s migration must default to enabled")
	}
}

func TestFeatureFlagsHonourExplicitFalse(t *testing.T) {
	restore := swapConfig(t)
	defer restore()

	raw := "cm-grasshopper:\n    features:\n        software_migration: false\n        k8s_migration: true\n"
	if err := yaml.Unmarshal([]byte(raw), &CMGrasshopperConfig); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if SoftwareMigrationEnabled() {
		t.Error("software_migration: false must disable software migration")
	}
	if !K8sMigrationEnabled() {
		t.Error("k8s_migration: true must keep k8s migration enabled")
	}
}

// A k8s-only deployment has no ansible playbooks and no software folders. Validating them
// anyway is what made the process refuse to start.
func TestCheckConfig_SkipsSoftwarePathsWhenDisabled(t *testing.T) {
	restore := swapConfig(t)
	defer restore()

	fillValidK8sConfig(t)
	disabled := false
	CMGrasshopperConfig.CMGrasshopper.Features.SoftwareMigration = &disabled

	if err := checkCMGrasshopperConfigFile(); err != nil {
		t.Errorf("software paths must not be validated when software migration is off: %v", err)
	}
}

func TestCheckConfig_ValidatesSoftwarePathsWhenEnabled(t *testing.T) {
	restore := swapConfig(t)
	defer restore()

	fillValidK8sConfig(t)

	if err := checkCMGrasshopperConfigFile(); err == nil {
		t.Error("expected the missing software temp_folder to be reported while enabled")
	}
}

func fillValidK8sConfig(t *testing.T) {
	t.Helper()

	cfg := &CMGrasshopperConfig.CMGrasshopper
	cfg.Listen.Port = "8084"
	cfg.Honeybee.ServerPort = "8081"
	cfg.Tumblebug.ServerPort = "1323"
	cfg.K8s.JobWorkerCount = 3
	cfg.K8s.JobLogFolder = t.TempDir()
}

func swapConfig(t *testing.T) func() {
	t.Helper()

	saved := CMGrasshopperConfig
	CMGrasshopperConfig = cmGrasshopperConfig{}

	return func() { CMGrasshopperConfig = saved }
}
