package model

type TBNodeInfo struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	Label        map[string]string `json:"label"`
	PublicIP     string            `json:"publicIP"`
	SSHPort      int               `json:"sshPort"`
	SSHKeyID     string            `json:"sshKeyId"`
	NodeUserName string            `json:"nodeUserName,omitempty"`
}

type TBInfraInfo struct {
	Node []TBNodeInfo `json:"node"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
