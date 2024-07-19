package model

type TBVMInfo struct {
	PublicIP      string `json:"publicIP"`
	SSHPort       string `json:"sshPort"`
	SSHKeyID      string `json:"sshKeyId"`
	VMUserAccount string `json:"vmUserAccount,omitempty"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
