package software

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"net"
	"strconv"
	"strings"
)

type Connection struct {
	Protocol       string
	LocalAddress   string
	ForeignAddress string
	PID            int
	Command        string
	ProgramName    string
}

func GetServicePID(client *ssh.Client, serviceName, password string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := sudoWrapper(fmt.Sprintf("systemctl show %s -p MainPID", serviceName), password)
	var out bytes.Buffer
	session.Stdout = &out

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("failed to run systemctl show: %v", err)
	}

	output := strings.TrimSpace(out.String())
	parts := strings.Split(output, "=")
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("no PID found for service %s (service may not be running)", serviceName)
	}

	return parts[1], nil
}

func GetListeningConnections(client *ssh.Client, password string) ([]Connection, error) {
	var connections []Connection

	pids, err := getPIDs(client, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get pid list: %v", err)
	}

	tcpConnections, err := readProcNetFile(client, password, "tcp", false, pids)
	if err != nil {
		return nil, fmt.Errorf("failed to read TCPv4 connections: %v", err)
	}
	connections = append(connections, tcpConnections...)

	tcp6Connections, err := readProcNetFile(client, password, "tcp6", true, pids)
	if err != nil {
		return nil, fmt.Errorf("failed to read TCPv6 connections: %v", err)
	}
	connections = append(connections, tcp6Connections...)

	udpConnections, err := readProcNetFile(client, password, "udp", false, pids)
	if err != nil {
		return nil, fmt.Errorf("failed to read UDPv4 connections: %v", err)
	}
	connections = append(connections, udpConnections...)

	udp6Connections, err := readProcNetFile(client, password, "udp6", true, pids)
	if err != nil {
		return nil, fmt.Errorf("failed to read UDPv6 connections: %v", err)
	}
	connections = append(connections, udp6Connections...)

	return connections, nil
}

func readProcNetFile(client *ssh.Client, password, protocol string, isIPv6 bool, pids []string) ([]Connection, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	filePath := fmt.Sprintf("/proc/net/%s", protocol)
	cmd := sudoWrapper(fmt.Sprintf("cat %s", filePath), password)
	var out bytes.Buffer
	session.Stdout = &out

	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", filePath, err)
	}

	var connections []Connection

	scanner := bufio.NewScanner(&out)
	if scanner.Scan() {
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)

			localAddr := hexToIPPort(fields[1], isIPv6)
			foreignAddr := hexToIPPort(fields[2], isIPv6)
			state := fields[3]

			if state == "0A" { // LISTEN state is represented as 0A in hex
				pid, err := findPIDByInode(client, password, fields[9], pids)
				if err != nil {
					continue
				}

				command := readCommandLine(client, password, pid)
				programName := readProgramName(client, password, pid)

				connections = append(connections, Connection{
					Protocol:       protocol,
					LocalAddress:   localAddr,
					ForeignAddress: foreignAddr,
					PID:            pid,
					Command:        command,
					ProgramName:    programName,
				})
			}
		}
	}

	return connections, nil
}

func getPIDs(client *ssh.Client, password string) ([]string, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := sudoWrapper("ls /proc | grep -E '^[0-9]+$'", password)
	var out bytes.Buffer
	session.Stdout = &out

	if err := session.Run(cmd); err != nil {
		return nil, fmt.Errorf("failed to list /proc directory: %v", err)
	}

	pids := strings.Split(strings.TrimSpace(out.String()), "\n")

	return pids, nil
}

func findPIDByInode(client *ssh.Client, password, inode string, pids []string) (int, error) {
	for _, pid := range pids {
		fdDir := fmt.Sprintf("/proc/%s/fd", pid)
		fdsSession, err := client.NewSession()
		if err != nil {
			continue
		}

		var fdsOut bytes.Buffer
		fdsSession.Stdout = &fdsOut

		fdCmd := sudoWrapper(fmt.Sprintf("ls -l %s", fdDir), password)
		if err := fdsSession.Run(fdCmd); err != nil {
			_ = fdsSession.Close()
			continue
		}

		_ = fdsSession.Close()

		fdsOutput := strings.TrimSpace(fdsOut.String())
		for _, line := range strings.Split(fdsOutput, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}

			link := fields[len(fields)-1]
			if strings.Contains(link, inode) {
				pidInt, err := strconv.Atoi(pid)
				if err != nil {
					continue
				}
				return pidInt, nil
			}
		}
	}

	return 0, fmt.Errorf("failed to find PID for inode: %s", inode)
}

func compareServicePorts(sourceClient, targetClient *ssh.Client, serviceName string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Starting service migration for package: %s\n", serviceName)

	migrationLogger.Printf(DEBUG, "Detecting source PID\n")
	sourcePID, err := GetServicePID(sourceClient, serviceName, sourceClient.SSHTarget.Password)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get PID on source: %v\n", err)
		return fmt.Errorf("failed to get PID on source: %v", err)
	}
	sourcePortPID, _ := strconv.Atoi(sourcePID)
	migrationLogger.Printf(DEBUG, "Source PID: %s\n", sourcePID)

	migrationLogger.Printf(DEBUG, "Detecting target PID\n")
	targetPID, err := GetServicePID(targetClient, serviceName, targetClient.SSHTarget.Password)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get PID on target: %v\n", err)
		return fmt.Errorf("failed to get PID on target: %v", err)
	}
	targetPortPID, _ := strconv.Atoi(targetPID)
	migrationLogger.Printf(DEBUG, "Target PID: %s\n", targetPID)

	migrationLogger.Printf(DEBUG, "Retrieving source listening connections\n")
	sourceConnections, err := GetListeningConnections(sourceClient, sourceClient.SSHTarget.Password)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get listening connections on source: %v\n", err)
		return fmt.Errorf("failed to get listening connections on source: %v", err)
	}
	sourceServiceConnections := filterConnectionsByPID(sourceConnections, sourcePortPID)
	migrationLogger.Printf(DEBUG, "Source listening connections:\n")
	for _, conn := range sourceServiceConnections {
		migrationLogger.Printf(DEBUG, "- Protocol: %s, Local Address: %s, Foreign Address: %s, PID: %d, Program: %s, Command: %s\n",
			conn.Protocol, conn.LocalAddress, conn.ForeignAddress, conn.PID, conn.ProgramName, conn.Command)
	}

	migrationLogger.Printf(DEBUG, "Retrieving target listening connections\n")
	targetConnections, err := GetListeningConnections(targetClient, targetClient.SSHTarget.Password)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get listening connections on target: %v\n", err)
		return fmt.Errorf("failed to get listening connections on target: %v", err)
	}
	targetServiceConnections := filterConnectionsByPID(targetConnections, targetPortPID)
	migrationLogger.Printf(DEBUG, "Target listening connections:\n")
	for _, conn := range targetServiceConnections {
		migrationLogger.Printf(DEBUG, "- Protocol: %s, Local Address: %s, Foreign Address: %s, PID: %d, Program: %s, Command: %s\n",
			conn.Protocol, conn.LocalAddress, conn.ForeignAddress, conn.PID, conn.ProgramName, conn.Command)
	}

	var mismatchedPorts []string
	for _, sourceConn := range sourceServiceConnections {
		var found bool
		for _, targetConn := range targetServiceConnections {
			if sourceConn.LocalAddress == targetConn.LocalAddress {
				migrationLogger.Printf(INFO, "Matching port found: %s %s\n",
					sourceConn.Protocol, sourceConn.LocalAddress)
				found = true
				break
			}
		}
		if !found {
			mismatchedPorts = append(mismatchedPorts, sourceConn.Protocol+" "+sourceConn.LocalAddress)
			migrationLogger.Printf(ERROR, "No matching port for %s %s on target\n",
				sourceConn.Protocol, sourceConn.LocalAddress)
		}
	}

	if len(mismatchedPorts) > 0 {
		return fmt.Errorf("some listen ports are not matched on target: %s", strings.Join(mismatchedPorts, ", "))
	}

	return nil
}

func filterConnectionsByPID(connections []Connection, pid int) []Connection {
	var filteredConnections []Connection
	for _, conn := range connections {
		if conn.PID == pid {
			filteredConnections = append(filteredConnections, conn)
		}
	}
	return filteredConnections
}

func hexToIPPort(hex string, isIPv6 bool) string {
	parts := strings.Split(hex, ":")
	ipHex := parts[0]
	portHex := parts[1]

	var ip string
	if isIPv6 {
		ip = parseIPv6Hex(ipHex)
	} else {
		ip = fmt.Sprintf("%d.%d.%d.%d",
			parseHexByte(ipHex[6:8]),
			parseHexByte(ipHex[4:6]),
			parseHexByte(ipHex[2:4]),
			parseHexByte(ipHex[0:2]),
		)
	}

	port, _ := strconv.ParseInt(portHex, 16, 32)
	return fmt.Sprintf("%s:%d", ip, port)
}

func parseIPv6Hex(hex string) string {
	ipSegs := make([]string, 8)
	for i := 0; i < 8; i++ {
		ipSegs[i] = hex[4*i : 4*(i+1)]
	}

	for i, seg := range ipSegs {
		ipSegs[i] = fmt.Sprintf("%x", parseHexSegment(seg))
	}

	ip := strings.Join(ipSegs, ":")
	compressedIP := net.ParseIP(ip).String() // This automatically compresses the zeros
	return compressedIP
}

func parseHexSegment(hex string) int {
	parsed, _ := strconv.ParseInt(hex, 16, 16)
	return int(parsed)
}

func parseHexByte(hex string) int {
	parsed, _ := strconv.ParseInt(hex, 16, 8)
	return int(parsed)
}

func readCommandLine(client *ssh.Client, password string, pid int) string {
	session, err := client.NewSession()
	if err != nil {
		return "Unknown"
	}
	defer func() {
		_ = session.Close()
	}()

	filePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmd := sudoWrapper(fmt.Sprintf("cat %s", filePath), password)
	var out bytes.Buffer
	session.Stdout = &out

	if err := session.Run(cmd); err != nil {
		return "Unknown"
	}

	return strings.ReplaceAll(strings.TrimSpace(out.String()), "\x00", " ")
}

func readProgramName(client *ssh.Client, password string, pid int) string {
	session, err := client.NewSession()
	if err != nil {
		return "Unknown"
	}
	defer func() {
		_ = session.Close()
	}()

	filePath := fmt.Sprintf("/proc/%d/comm", pid)
	cmd := sudoWrapper(fmt.Sprintf("cat %s", filePath), password)
	var out bytes.Buffer
	session.Stdout = &out

	if err := session.Run(cmd); err != nil {
		return "Unknown"
	}

	return strings.ReplaceAll(strings.TrimSpace(out.String()), "\x00", " ")
}

func listenPortsValidator(sourceClient *ssh.Client, targetClient *ssh.Client, serviceName string, migrationLogger *Logger) error {
	err := compareServicePorts(sourceClient, targetClient, serviceName, migrationLogger)
	if err != nil {
		migrationLogger.Printf(ERROR, "Error during comparison: %v\n", err)
		return err
	}

	return nil
}
