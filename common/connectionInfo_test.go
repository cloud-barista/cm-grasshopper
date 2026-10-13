package common

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"

	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// encryptLikeHoneybee mirrors cm-honeybee's chunked RSA-OAEP(SHA-512) + base64 encoding.
func encryptLikeHoneybee(t *testing.T, pub *rsa.PublicKey, plaintext string) string {
	t.Helper()
	if plaintext == "" {
		return ""
	}
	hash := sha512.New()
	maxLen := pub.Size() - 2*hash.Size() - 2
	msg := []byte(plaintext)
	var out bytes.Buffer
	for offset := 0; offset < len(msg); offset += maxLen {
		end := offset + maxLen
		if end > len(msg) {
			end = len(msg)
		}
		ct, err := rsa.EncryptOAEP(hash, rand.Reader, pub, msg[offset:end], nil)
		if err != nil {
			t.Fatalf("EncryptOAEP: %v", err)
		}
		out.Write(ct)
	}
	return base64.StdEncoding.EncodeToString(out.Bytes())
}

func TestConnectionInfoDecryptSecrets(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	saved := HoneybeePrivateKey
	HoneybeePrivateKey = key
	t.Cleanup(func() { HoneybeePrivateKey = saved })

	// Longer than one RSA chunk so multi-chunk decryption is exercised.
	kubeconfig := "apiVersion: v1\nkind: Config\nclusters:\n- cluster:\n    server: https://192.0.2.10:6443\n" +
		"    certificate-authority-data: " + strings.Repeat("Q", 300) + "\n  name: onprem\n"

	t.Run("ssh connection without kubeconfig", func(t *testing.T) {
		ci := &honeybee.ConnectionInfo{
			User:       encryptLikeHoneybee(t, &key.PublicKey, "cb-user"),
			Password:   encryptLikeHoneybee(t, &key.PublicKey, "secret"),
			PrivateKey: encryptLikeHoneybee(t, &key.PublicKey, "-"),
		}
		got, err := ConnectionInfoDecryptSecrets(ci)
		if err != nil {
			t.Fatalf("ConnectionInfoDecryptSecrets() error = %v", err)
		}
		if got.User != "cb-user" || got.Password != "secret" || got.PrivateKey != "-" || got.Kubeconfig != "" {
			t.Fatalf("unexpected decrypted fields: %+v", got)
		}
	})

	t.Run("k8s connection with kubeconfig only", func(t *testing.T) {
		ci := &honeybee.ConnectionInfo{
			Kubeconfig: encryptLikeHoneybee(t, &key.PublicKey, kubeconfig),
		}
		got, err := ConnectionInfoDecryptSecrets(ci)
		if err != nil {
			t.Fatalf("ConnectionInfoDecryptSecrets() error = %v", err)
		}
		if got.Kubeconfig != kubeconfig {
			t.Fatalf("Kubeconfig = %q, want %q", got.Kubeconfig, kubeconfig)
		}
		if got.User != "" || got.Password != "" || got.PrivateKey != "" {
			t.Fatalf("SSH fields should stay empty: %+v", got)
		}
	})

	t.Run("invalid base64 kubeconfig", func(t *testing.T) {
		ci := &honeybee.ConnectionInfo{Kubeconfig: "%%%"}
		if _, err := ConnectionInfoDecryptSecrets(ci); err == nil {
			t.Fatal("expected error for invalid base64 kubeconfig")
		}
	})
}
