package software

import (
	"embed"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"strings"
)

//go:embed scripts/*.sh
var scriptsFS embed.FS

func executeScript(client *ssh.Client, scriptName string, args ...string) (string, error) {
	script, err := getScript(scriptName)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	var cmd string
	if len(args) > 0 {
		cmd = fmt.Sprintf("cat << 'EOF' | sh -s %s\n%s\nEOF", strings.Join(args, " "), script)
	} else {
		cmd = fmt.Sprintf("cat << 'EOF' | sh\n%s\nEOF", script)
	}

	output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	return string(output), err
}
