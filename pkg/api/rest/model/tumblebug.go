package model

type CSPViewVMDetail struct {
	PublicIP string `json:"PublicIP"`
}

type TBVMInfo struct {
	PublicIP        string          `json:"publicIP"`
	SSHPort         string          `json:"sshPort"`
	SSHKeyID        string          `json:"sshKeyId"`
	VMUserAccount   string          `json:"vmUserAccount,omitempty"`
	CSPViewVMDetail CSPViewVMDetail `json:"cspViewVmDetail"`
}

type TBSSHKeyInfo struct {
	PrivateKey string `json:"privateKey,omitempty"`
}
