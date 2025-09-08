package software

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
)

//go:embed scripts/*.sh
var scriptsFS embed.FS

func executeScript(client *ssh.Client, migrationLogger *Logger, scriptName string, args ...string) ([]byte, error) {
	script, err := getScript(scriptName)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSessionWithRetry()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = session.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				migrationLogger.Printf(DEBUG, "Sending SSH keep-alive for script: %s\n", scriptName)

				keepAliveSession, err := client.NewSessionWithRetry()
				if err != nil {
					migrationLogger.Printf(DEBUG, "Keep-alive session creation failed: %v\n", err)
					continue
				}

				_, _ = keepAliveSession.CombinedOutput("echo keepalive")
				_ = keepAliveSession.Close()
			}
		}
	}()

	var cmd string
	if len(args) > 0 {
		cmd = fmt.Sprintf("cat << 'EOF' | sh -s %s\n%s\nEOF", strings.Join(args, " "), script)
	} else {
		cmd = fmt.Sprintf("cat << 'EOF' | sh\n%s\nEOF", script)
	}
	wrappedCmd := sudoWrapper(cmd, client.SSHTarget.Password)

	migrationLogger.Printf(DEBUG, "Executing script with keep-alive enabled\n")
	output, err := session.CombinedOutput(wrappedCmd)

	cancel()

	return output, err
}
