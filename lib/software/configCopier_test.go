package software

import (
	"strings"
	"testing"
)

// Ubuntu's /etc/exports after "apt-get install nfs-kernel-server" plus the share
// line of the source host. It must be treated as an NFS exports file so that
// the shared directory is created on the target.
func TestIsNfsExportContentDetectsSharedDirectory(t *testing.T) {
	content := `# /etc/exports: the access control list for filesystems which may be exported
#		to NFS clients.  See exports(5).
#
# Example for NFSv2 and NFSv3:
# /srv/homes       hostname1(rw,sync,no_subtree_check) hostname2(ro,sync,no_subtree_check)
#
/nfs/share *(rw,sync,no_subtree_check,no_root_squash)
`
	if !isNfsExportContent(content) {
		t.Fatalf("isNfsExportContent() = false, want true for %q", content)
	}
}

func TestIsNfsExportContentIgnoresCommentOnlyFile(t *testing.T) {
	content := "# /etc/exports: the access control list\n# /srv/homes hostname1(rw,sync)\n"
	if isNfsExportContent(content) {
		t.Fatalf("isNfsExportContent() = true, want false for comment-only file")
	}
}

// config_finder.sh marks /etc/exports as [Custom]; copyConfigFiles only skips
// [Unmodified] entries, so the file must keep a status other than that.
func TestParseConfigLineKeepsCustomStatus(t *testing.T) {
	conf := parseConfigLine("/etc/exports [Custom]", &Logger{})
	if conf.Path != "/etc/exports" || conf.Status != "Custom" {
		t.Fatalf("parseConfigLine() = %+v, want Path=/etc/exports Status=Custom", conf)
	}
	if strings.Contains(conf.Status, "Unmodified") {
		t.Fatalf("status %q would be skipped by copyConfigFiles", conf.Status)
	}
}

// /etc/exports is not in the file list of the NFS server package on either
// distribution (Ubuntu creates it with ucf, RHEL ships it in "setup"), so the
// finder has to name it explicitly.
func TestConfigFinderListsNfsExportsForServerPackages(t *testing.T) {
	script, err := getScript("config_finder.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"nfs-kernel-server /etc/exports /etc/exports.d/*",
		"nfs-utils /etc/exports /etc/exports.d/*",
	} {
		if !strings.Contains(script, "\n"+line+"\n") {
			t.Errorf("config_finder.sh has no well-known config line %q", line)
		}
	}
}
