package software

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type SystemType int

const (
	Unknown SystemType = iota
	Debian
	RHEL
)

type ServiceInfo struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	State   string `json:"state,omitempty"`
}

func cleanServiceName(name string) string {
	cleaned := name
	cleaned = strings.ReplaceAll(cleaned, "├─", "")
	cleaned = strings.ReplaceAll(cleaned, "└─", "")
	cleaned = strings.ReplaceAll(cleaned, "│", "")
	cleaned = strings.ReplaceAll(cleaned, "●", "")
	cleaned = strings.ReplaceAll(cleaned, "○", "")
	cleaned = strings.TrimSpace(cleaned)

	fields := strings.Fields(cleaned)
	if len(fields) > 0 {
		cleaned = fields[0]
	}

	parts := strings.Split(cleaned, "/")
	if len(parts) > 0 {
		cleaned = parts[len(parts)-1]
	}

	return cleaned
}

func getRealServiceName(client *ssh.Client, name string, migrationLogger *Logger) (string, error) {
	var session *gossh.Session
	var err error

	session, err = client.NewSessionWithRetry()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
		return name, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := fmt.Sprintf("systemctl show -p Id --value %s", name)
	output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get real service name for %s: %s\n", name, string(output))
		return name, err
	}

	realName := strings.TrimSpace(string(output))
	if realName != "" && realName != name {
		realName = strings.TrimSuffix(realName, ".service")

		verifySession, err := client.NewSessionWithRetry()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create verification session: %v\n", err)
			return name, nil
		}
		defer func() {
			_ = verifySession.Close()
		}()

		verifyCmd := fmt.Sprintf("systemctl show %s -p LoadState --value", realName)
		verifyOutput, err := verifySession.CombinedOutput(sudoWrapper(verifyCmd, client.SSHTarget.Password))
		if err == nil && strings.TrimSpace(string(verifyOutput)) == "loaded" {
			migrationLogger.Printf(INFO, "Verified real service name: %s\n", realName)
			return realName, nil
		}

		migrationLogger.Printf(WARN, "Real service name %s exists but not loaded\n", realName)
	}

	migrationLogger.Printf(DEBUG, "No alternative name found for service: %s\n", name)
	return name, nil
}

func getServiceInfo(client *ssh.Client, name string, migrationLogger *Logger) ServiceInfo {
	migrationLogger.Printf(INFO, "Getting service info for: %s\n", name)

	info := ServiceInfo{
		Name: name,
	}

	var session *gossh.Session
	var err error

	session, err = client.NewSessionWithRetry()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
		return info
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := fmt.Sprintf("systemctl is-active %s", name)
	if output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password)); err == nil {
		info.Active = strings.TrimSpace(string(output)) == "active"
		migrationLogger.Printf(DEBUG, "Service %s active status: %v\n", name, info.Active)
	}

	session, err = client.NewSessionWithRetry()
	if err == nil {
		defer func() {
			_ = session.Close()
		}()
		cmd = fmt.Sprintf("systemctl is-enabled %s", name)
		if output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password)); err == nil {
			state := strings.TrimSpace(string(output))
			info.State = state
			migrationLogger.Printf(DEBUG, "Service %s state: %s\n", name, state)

			switch state {
			case "enabled", "enabled-runtime":
				info.Enabled = true
			case "static":
				newSession, err := client.NewSessionWithRetry()
				if err == nil {
					defer func() {
						_ = newSession.Close()
					}()
					cmd = fmt.Sprintf("systemctl show -p WantedBy --value %s", name)
					if output, err := newSession.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password)); err == nil && len(output) > 0 {
						info.Enabled = true
					}
				}
			case "alias":
				if realName, err := getRealServiceName(client, name, migrationLogger); err == nil {
					info.Name = realName
					newSession, err := client.NewSessionWithRetry()
					if err == nil {
						defer func() {
							_ = newSession.Close()
						}()
						cmd = fmt.Sprintf("systemctl is-enabled %s", realName)
						if output, err := newSession.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password)); err == nil {
							state = strings.TrimSpace(string(output))
							info.State = state
							info.Enabled = state == "enabled"
						}
					}
				}
			}
			migrationLogger.Printf(DEBUG, "Service %s enabled status: %v\n", name, info.Enabled)
		}
	}

	migrationLogger.Printf(INFO, "Completed service info gathering for %s\n", name)
	return info
}

func isSystemService(name string) bool {
	prefixes := []string{
		"sysinit.",
		"systemd-",
		"dbus-",
		"polkit-",
		"network-",
		"udev",
		"plymouth-",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
func findPackageRelatedServices(client *ssh.Client, packageName string) ([]string, error) {
	cmd := fmt.Sprintf("find /lib/systemd/system /usr/lib/systemd/system -name '*%s*.service'", packageName)
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(sudoWrapper(cmd, client.SSHTarget.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to find services related to package: %v, stderr: %s", err, stderr.String())
	}

	var services []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		serviceName := cleanServiceName(line)
		serviceName = strings.TrimSuffix(serviceName, ".service")
		services = append(services, serviceName)
	}

	cmd = fmt.Sprintf("systemctl list-units --type=service --all | grep '%s.service'", packageName)
	session, err = client.NewSessionWithRetry()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	stdout.Reset()
	stderr.Reset()
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(sudoWrapper(cmd, client.SSHTarget.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to list active services: %v, stderr: %s", err, stderr.String())
	}

	scanner = bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		serviceName := cleanServiceName(line)
		serviceName = strings.TrimSuffix(serviceName, ".service")
		if !contains(services, serviceName) {
			services = append(services, serviceName)
		}
	}

	return services, nil
}

func findServices(client *ssh.Client, name string, migrationLogger *Logger) []ServiceInfo {
	migrationLogger.Printf(INFO, "Finding services for package: %s\n", name)

	var services []ServiceInfo
	var foundServiceNames []string
	var filteredServices []string
	processedServices := make(map[string]bool)
	var err error

	foundServiceNames, err = findPackageRelatedServices(client, name)
	if err != nil {
		migrationLogger.Printf(WARN, "Failed to find services for package: %v\n", err)
		migrationLogger.Printf(INFO, "Trying to find real service name for package: %s\n", name)
		realName, err := getRealServiceName(client, name, migrationLogger)
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to find real service name for package: %v\n", err)
		}
		foundServiceNames = append(foundServiceNames, realName)
	}

	for _, serviceName := range foundServiceNames {
		if !processedServices[serviceName] && !isSystemService(serviceName) {
			filteredServices = append(filteredServices, serviceName)
			processedServices[serviceName] = true
		}
	}

	migrationLogger.Printf(INFO, "Found %d relevant services for package\n", len(filteredServices))

	for _, serviceName := range filteredServices {
		info := getServiceInfo(client, serviceName, migrationLogger)
		services = append(services, info)
	}

	migrationLogger.Printf(INFO, "Completed service discovery with %d services\n", len(services))
	return services
}

func getSystemType(client *ssh.Client, migrationLogger *Logger) (SystemType, error) {
	var session *gossh.Session
	var err error

	// Retry session creation up to 3 times
	for retry := 0; retry < 3; retry++ {
		session, err = client.NewSessionWithRetry()
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to create SSH session (attempt %d/3): %v\n", retry+1, err)
			if retry < 2 {
				time.Sleep(time.Second * 2) // Wait 2 seconds before retry
				continue
			}
			migrationLogger.Printf(ERROR, "Failed to create SSH session after 3 attempts: %v\n", err)
			return Unknown, fmt.Errorf("failed to create session: %v", err)
		}
		break
	}

	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput(sudoWrapper("cat /etc/os-release", client.SSHTarget.Password))
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to read os-release: %s\n", string(output))
		return Unknown, fmt.Errorf("failed to read os-release: %s", string(output))
	}

	var id string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			id = strings.ToLower(id)
			break
		}
	}

	migrationLogger.Printf(INFO, "Detected system ID: %s\n", id)

	switch id {
	case "debian", "ubuntu":
		return Debian, nil
	case "rhel", "centos", "rocky", "almalinux", "fedora", "ol":
		return RHEL, nil
	default:
		migrationLogger.Printf(ERROR, "Unsupported system type: %s\n", id)
		return Unknown, fmt.Errorf("unsupported system: %s", id)
	}
}

func serviceMigrator(sourceClient *ssh.Client, targetClient *ssh.Client, packageName, uuid string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Starting service migration for package: %s (UUID: %s)\n", packageName, uuid)

	services := findServices(sourceClient, packageName, migrationLogger)
	if len(services) == 0 {
		migrationLogger.Printf(WARN, "No services found for package: %s\n", packageName)
		return nil
	}

	migrationLogger.Printf(INFO, "Stopping services in dependency order\n")
	for _, service := range services {
		if service.Active {
			migrationLogger.Printf(INFO, "Stopping service: %s\n", service.Name)
			session, err := targetClient.NewSessionWithRetry()
			if err != nil {
				migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
				return fmt.Errorf("failed to create session: %v", err)
			}

			cmd := fmt.Sprintf("systemctl stop %s", service.Name)
			if output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password)); err != nil {
				migrationLogger.Printf(WARN, "Failed to stop service %s: %v\nOutput: %s\n",
					service.Name, err, string(output))
			}
			_ = session.Close()
		}
	}

	migrationLogger.Printf(INFO, "Setting service enable/disable states\n")
	for _, service := range services {
		session, err := targetClient.NewSessionWithRetry()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
			return fmt.Errorf("failed to create session: %v", err)
		}

		var cmd string
		if service.Enabled {
			migrationLogger.Printf(INFO, "Enabling service: %s\n", service.Name)
			cmd = fmt.Sprintf("systemctl enable %s", service.Name)
		} else {
			migrationLogger.Printf(INFO, "Disabling service: %s\n", service.Name)
			cmd = fmt.Sprintf("systemctl disable %s", service.Name)
		}

		if output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password)); err != nil {
			migrationLogger.Printf(WARN, "Failed to set enable state for %s: %v\nOutput: %s\n",
				service.Name, err, string(output))
		}
		_ = session.Close()
	}

	migrationLogger.Printf(INFO, "Starting services in reverse dependency order\n")
	for i := len(services) - 1; i >= 0; i-- {
		service := services[i]
		if service.Active {
			migrationLogger.Printf(INFO, "Starting service: %s\n", service.Name)

			session, err := targetClient.NewSessionWithRetry()
			if err != nil {
				migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
				return fmt.Errorf("failed to create session: %v", err)
			}

			cmd := fmt.Sprintf("systemctl start %s", service.Name)
			if output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password)); err != nil {
				migrationLogger.Printf(ERROR, "Failed to start service %s: %v\nOutput: %s\n",
					service.Name, err, string(output))
				_ = session.Close()
				return fmt.Errorf("failed to start service %s: %v", service.Name, err)
			}

			cmd = fmt.Sprintf("systemctl is-active %s", service.Name)
			output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password))
			if err != nil || strings.TrimSpace(string(output)) != "active" {
				migrationLogger.Printf(ERROR, "Service %s failed to start properly. Status: %s\n",
					service.Name, strings.TrimSpace(string(output)))
				_ = session.Close()
				return fmt.Errorf("service %s failed to start properly", service.Name)
			}
			_ = session.Close()

			migrationLogger.Printf(INFO, "Successfully started service: %s\n", service.Name)
		}
	}

	migrationLogger.Printf(INFO, "Verifying final states\n")
	var verificationErrors []string

	for _, service := range services {
		var issues []string

		session, err := targetClient.NewSessionWithRetry()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
			return fmt.Errorf("failed to create session: %v", err)
		}

		cmd := fmt.Sprintf("systemctl is-active %s", service.Name)
		output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password))
		if err == nil {
			isActive := strings.TrimSpace(string(output)) == "active"
			if isActive != service.Active {
				issue := fmt.Sprintf("active state mismatch (expected: %v, got: %v)",
					service.Active, isActive)
				issues = append(issues, issue)
				migrationLogger.Printf(WARN, "Service %s: %s\n", service.Name, issue)
			}
		}
		_ = session.Close()

		session, err = targetClient.NewSessionWithRetry()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
			return fmt.Errorf("failed to create session: %v", err)
		}

		cmd = fmt.Sprintf("systemctl is-enabled %s", service.Name)
		output, err = session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password))
		if err == nil {
			isEnabled := strings.TrimSpace(string(output)) == "enabled"
			if isEnabled != service.Enabled {
				issue := fmt.Sprintf("enabled state mismatch (expected: %v, got: %v)",
					service.Enabled, isEnabled)
				issues = append(issues, issue)
				migrationLogger.Printf(WARN, "Service %s: %s\n", service.Name, issue)
			}
		}
		_ = session.Close()

		err = listenPortsValidator(sourceClient, targetClient, service.Name, migrationLogger)
		if err != nil {
			issues = append(issues, err.Error())
		}

		if len(issues) > 0 {
			verificationError := fmt.Sprintf("Service %s has issues: %v", service.Name, issues)
			verificationErrors = append(verificationErrors, verificationError)
			migrationLogger.Printf(WARN, "Verification failed for service %s: %v\n", service.Name, issues)
		} else {
			migrationLogger.Printf(INFO, "Successfully verified service: %s\n", service.Name)
		}
	}

	if len(verificationErrors) > 0 {
		migrationLogger.Printf(ERROR, "Service migration completed with verification errors:\n%s\n",
			strings.Join(verificationErrors, "\n"))
		return fmt.Errorf("service migration completed with verification errors: %s",
			strings.Join(verificationErrors, "; "))
	}

	migrationLogger.Printf(INFO, "Successfully completed service migration for package: %s\n", packageName)
	return nil
}
