package model

type TBVMInfo struct {
	Id         string            `json:"id"`
	Label      map[string]string `json:"label"`
	PublicIP   string            `json:"publicIP"`
	SSHPort    string            `json:"sshPort"`
	SSHKeyID   string            `json:"sshKeyId"`
	VMUserName string            `json:"vmUserName,omitempty"`
}

type TBMCIInfo struct {
	VM []TBVMInfo `json:"vm"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
