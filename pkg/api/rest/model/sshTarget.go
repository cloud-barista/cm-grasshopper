package model

type SSHTarget struct {
	IP         string `json:"ip"`
	Port       uint   `json:"port"`
	UseKeypair bool   `json:"use_keypair"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}
