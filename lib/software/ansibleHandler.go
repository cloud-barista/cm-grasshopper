package software

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/iputil"
	"github.com/jollaman999/utils/logger"
	"github.com/mholt/archiver/v3"
	cp "github.com/otiai10/copy"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ansibleConfigPath string
var inventoryFileName = "inventory.ini"
var logFileName = "ansible_output.log"

func getAnsiblePlaybookFolderPath(softwareID string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath, softwareID))
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return "", err
	}
	return abs, nil
}

func getTemporaryAnsiblePlaybookFolderPath(executionID string, softwareID string) (string, error) {
	absSource, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath, softwareID))
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return "", err
	}

	absDest, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Software.TempFolder, executionID))
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return "", err
	}

	err = cp.Copy(absSource, absDest)
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return "", err
	}

	return absDest, nil
}

func getAnsiblePlaybookLogPath(executionID string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Software.LogFolder, executionID))
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return "", err
	}
	return abs, nil
}

// checkRootDirectory: If archive file has one folder, move inside items to the top.
func checkRootDirectory(pwd string, files []os.DirEntry) bool {
	if files == nil {
		return false
	}

	if len(files) == 1 && files[0].IsDir() {
		originPath := filepath.Join(pwd, files[0].Name())
		tempPath := pwd + "_temp"

		err := moveDir(originPath, tempPath)
		if err != nil {
			deleteDir(pwd)
			return false
		}

		deleteDir(pwd)

		err = moveDir(tempPath, pwd)
		if err != nil {
			deleteDir(tempPath)
			return false
		}

		files, err = getFiles(pwd)
		if err != nil {
			return false
		}

		return checkRootDirectory(pwd, files)
	}

	return true
}

func findPlaybookFile(id string) (string, error) {
	var yamlFound bool
	var foundPlaybookFile string

	pwd, err := getAnsiblePlaybookFolderPath(id)
	if err != nil {
		return "", err
	}

	files, err := getFiles(pwd)
	if err != nil {
		return "", err
	}

	if !checkRootDirectory(pwd, files) {
		return "", errors.New("ANSIBLE: failed to check root directory in the archive file")
	}

	files, err = getFiles(pwd)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := strings.ToLower(file.Name())
		if getFileExtension(fileName) == ".yaml" || getFileExtension(fileName) == ".yml" {
			if yamlFound {
				return "", errors.New("ANSIBLE: found multiple yaml files")
			}
			yamlFound = true
			foundPlaybookFile = fileName
		}
	}

	if yamlFound {
		return foundPlaybookFile, nil
	}

	return "", errors.New("ANSIBLE: failed to find any yaml files")
}

func decompressFile(filePath string, destDir string) error {
	err := archiver.Unarchive(filePath, destDir)
	if err != nil {
		return errors.New("ANSIBLE: failed to decompress file (File: " + filePath + ", Error: " + err.Error() + ")")
	}

	return nil
}

func SavePlaybook(id string, playbookArchiveFile string) (sizeString string, err error) {
	pwd := ""
	sizeString = "0B"

	pwd, err = getAnsiblePlaybookFolderPath(id)
	if err != nil {
		return sizeString, err
	}

	err = decompressFile(playbookArchiveFile, pwd)
	if err != nil {
		logger.Logger.Println(logger.ERROR, true, err)
		return sizeString, err
	}

	err = os.Remove(playbookArchiveFile)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return sizeString, err
	}

	_, err = findPlaybookFile(id)
	if err != nil {
		logger.Logger.Println(logger.ERROR, true, err)
		return sizeString, err
	}

	size, err := getFolderSize(pwd)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return sizeString, err
	}

	sizeString = formatSize(size)

	return sizeString, err
}

func inventoryFileCreate(pwd string, sshTarget model.SSHTarget) error {
	fileContent := bytes.NewBufferString("[all]\n")

	ip := iputil.CheckValidIP(sshTarget.IP)
	if ip == nil {
		return errors.New("invalid IP address")
	}

	if sshTarget.Username == "" {
		return errors.New("username is not provided")
	}

	if sshTarget.Port < 1 || sshTarget.Port > 65535 {
		return errors.New("invalid port number")
	}

	hostConfig := fmt.Sprintf("%s ansible_port=%d", sshTarget.IP, sshTarget.Port)

	if sshTarget.UseKeypair {
		privateKey := sshTarget.PrivateKey
		if privateKey == "" {
			return errors.New("private_key is not provided with use_keypair=true")
		}

		privateKeyFilePath := filepath.Join(pwd, "private_key.pem")
		err := os.WriteFile(privateKeyFilePath, []byte(privateKey), 0600)
		if err != nil {
			return err
		}

		hostConfig += fmt.Sprintf(" ansible_ssh_private_key_file=%s ansible_ssh_user=%s", privateKeyFilePath, sshTarget.Username)
	} else {
		if sshTarget.Password == "" {
			return errors.New("password is not provided with use_keypair=false")
		}

		hostConfig += fmt.Sprintf(" ansible_ssh_user=%s ansible_ssh_pass=%s", sshTarget.Username, sshTarget.Password)
	}

	_, err := fileContent.WriteString(hostConfig + "\n")
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(pwd, inventoryFileName), fileContent.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}

func playbookHostsReplacer(playbookFilePath string) error {
	file, err := os.Open(playbookFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	var newPlaybookContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		lineCompressed := strings.ReplaceAll(strings.ReplaceAll(line, " ", ""), "\t", "")
		if strings.Contains(lineCompressed, "hosts:") {
			idx := strings.Index(line, "hosts")
			line = line[:idx] + "hosts: all\n"
		}
		newPlaybookContent.WriteString(line + "\n")
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	err = fileutil.WriteFile(playbookFilePath, newPlaybookContent.String())
	if err != nil {
		return err
	}

	return nil
}

func checkAnsibleConfig() error {
	if fileutil.IsExist(ansibleConfigPath) {
		return nil
	}

	ansibleConfigPath = filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath, "ansible.cfg")
	var configFileContent = "[defaults]\n" +
		"command_warnings = false\n" +
		"deprecation_warnings=false\n" +
		"ask_pass = false\n" +
		"callbacks_enabled = profile_tasks, profile_roles\n" +
		"host_key_checking = false\n" +
		"display_failed_stderr=yes\n" +
		"\n" +
		"[privilege_escalation]\n" +
		"become = true\n" +
		"become_method = sudo\n" +
		"become_user = root\n" +
		"become_ask_pass = false\n" +
		"\n" +
		"[ssh_connection]\n" +
		"pipelining = true\n" +
		"ssh_args = -o ControlMaster=no -o ControlPersist=no\n"

	err := fileutil.WriteFile(ansibleConfigPath, configFileContent)
	if err != nil {
		return err
	}

	return nil
}

func runPlaybook(executionID string, softwareID string, sshTarget model.SSHTarget) error {
	err := checkAnsibleConfig()
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}

	pwd, err := getTemporaryAnsiblePlaybookFolderPath(executionID, softwareID)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}
	defer func() {
		deleteDir(pwd)
	}()

	playbookFile, err := findPlaybookFile(softwareID)
	if err != nil {
		logger.Logger.Println(logger.ERROR, true, err)
		return err
	}
	playbookFilePath := filepath.Join(pwd, playbookFile)

	err = playbookHostsReplacer(playbookFilePath)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}

	inventoryFilePath := filepath.Join(pwd, inventoryFileName)
	err = inventoryFileCreate(pwd, sshTarget)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}

	cmd := exec.Command("ansible-playbook", playbookFilePath, "-i", inventoryFilePath)
	cmd.Env = []string{
		"ANSIBLE_CONFIG=" + ansibleConfigPath,
	}

	logPath, err := getAnsiblePlaybookLogPath(executionID)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}

	err = fileutil.CreateDirIfNotExist(logPath)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}

	logFile, err := os.OpenFile(filepath.Join(logPath, logFileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
	}
	defer func() {
		_ = logFile.Close()
	}()

	mw := io.MultiWriter(os.Stdout, logFile)

	cmd.Stdout = mw
	cmd.Stderr = mw

	err = cmd.Run()
	if err != nil {
		errMsg := "ANSIBLE: Failed to run the playbook. (ID: " + softwareID + ", Error: " + err.Error() + ")"
		logger.Logger.Println(logger.ERROR, true, errMsg)
	}

	return nil
}

func DeletePlaybook(id string) error {
	playbookPath, err := getAnsiblePlaybookFolderPath(id)
	if err != nil {
		errMsg := "ANSIBLE: " + err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return err
	}
	deleteDir(playbookPath)

	return nil
}
