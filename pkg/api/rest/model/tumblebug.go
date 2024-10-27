package model

type TBVMInfo struct {
	PublicIP   string `json:"publicIP"`
	SSHPort    string `json:"sshPort"`
	SSHKeyID   string `json:"sshKeyId"`
	VMUserName string `json:"vmUserName,omitempty"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
