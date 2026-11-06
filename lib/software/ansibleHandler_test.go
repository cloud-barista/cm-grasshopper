package software

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	cp "github.com/otiai10/copy"
)

// TestRunPlaybookConcurrentTargetsIsolated runs the package playbook for two
// targets of the same execution at once, as MigrateSoftware does (one goroutine
// per target, one execution ID). Each ansible-playbook run must see only its own
// target host and its own package; a shared working directory let one target's
// run install another target's package on another host while its own package was
// reported finished.
func TestRunPlaybookConcurrentTargetsIsolated(t *testing.T) {
	root := t.TempDir()
	playbookRoot := filepath.Join(root, "playbook")
	if err := cp.Copy(filepath.Join("..", "..", "playbook"), playbookRoot); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(playbookRoot, "ansible.cfg"))
	tempFolder := filepath.Join(root, "temp")
	logFolder := filepath.Join(root, "log")
	recordDir := filepath.Join(root, "record")
	binDir := filepath.Join(root, "bin")
	for _, d := range []string{tempFolder, logFolder, recordDir, binDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// A stand-in for ansible-playbook that records the inventory and the package
	// variables it was handed, then lingers so the two runs overlap.
	fake := "#!/bin/sh\n" +
		"d=$(/usr/bin/dirname \"$1\")\n" +
		"{ /bin/cat \"$3\"; /bin/cat \"$d/roles/role/vars/main.yml\"; } > " + recordDir + "/$$ 2>&1\n" +
		"/bin/sleep 0.5\n"
	if err := os.WriteFile(filepath.Join(binDir, "ansible-playbook"), []byte(fake), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	saved := config.CMGrasshopperConfig
	savedCfgPath := ansibleConfigPath
	t.Cleanup(func() {
		config.CMGrasshopperConfig = saved
		ansibleConfigPath = savedCfgPath
	})
	config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath = playbookRoot
	config.CMGrasshopperConfig.CMGrasshopper.Software.TempFolder = tempFolder
	config.CMGrasshopperConfig.CMGrasshopper.Software.LogFolder = logFolder
	ansibleConfigPath = ""

	targets := map[string]string{
		"192.0.2.11": "mariadb-server",
		"192.0.2.12": "nginx",
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(targets))
	for ip, pkg := range targets {
		wg.Add(1)
		go func(ip, pkg string) {
			defer wg.Done()
			errs <- runPlaybook("ex-1", "package", pkg, &model.SSHTarget{
				IP: ip, Port: 22, Username: "ubuntu", Password: "pw",
			})
		}(ip, pkg)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("runPlaybook: %v", err)
		}
	}

	if left, _ := os.ReadDir(tempFolder); len(left) != 0 {
		t.Errorf("temp folder not cleaned up, %d entries left", len(left))
	}

	records, err := os.ReadDir(recordDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != len(targets) {
		t.Fatalf("ansible-playbook ran %d times, want %d", len(records), len(targets))
	}
	for _, r := range records {
		b, err := os.ReadFile(filepath.Join(recordDir, r.Name()))
		if err != nil {
			t.Fatal(err)
		}
		seen := string(b)
		var hosts []string
		for ip := range targets {
			if strings.Contains(seen, ip+" ") {
				hosts = append(hosts, ip)
			}
		}
		if len(hosts) != 1 {
			t.Errorf("run saw hosts %v, want exactly one:\n%s", hosts, seen)
			continue
		}
		want := targets[hosts[0]]
		for ip, pkg := range targets {
			if ip != hosts[0] && strings.Contains(seen, "  - "+pkg+"\n") {
				t.Errorf("run for %s (%s) was also handed %s of %s:\n%s", hosts[0], want, pkg, ip, seen)
			}
		}
		if !strings.Contains(seen, "  - "+want+"\n") {
			t.Errorf("run for %s was not handed its own package %s:\n%s", hosts[0], want, seen)
		}
	}
}
