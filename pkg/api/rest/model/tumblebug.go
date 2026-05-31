package model

type TBVMInfo struct {
	Id         string            `json:"id"`
	Label      map[string]string `json:"label"`
	PublicIP   string            `json:"publicIP"`
	SSHPort    int               `json:"sshPort"`
	SSHKeyID   string            `json:"sshKeyId"`
	VMUserName string            `json:"nodeUserName,omitempty"`
}

type TBMCIInfo struct {
	VM []TBVMInfo `json:"node"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
