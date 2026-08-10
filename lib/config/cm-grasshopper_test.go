package config

import (
	"testing"
)

// Software and k8s config are always validated (both subsystems are always enabled).
// A missing software temp_folder must be reported.
func TestCheckConfig_ValidatesSoftwarePaths(t *testing.T) {
	restore := swapConfig(t)
	defer restore()

	fillValidK8sConfig(t)

	if err := checkCMGrasshopperConfigFile(); err == nil {
		t.Error("expected the missing software temp_folder to be reported")
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
