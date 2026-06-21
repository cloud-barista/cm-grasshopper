package tumblebug

import (
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

// EKS-style kubeconfig: exec plugin shelling out to aws-iam-authenticator (as returned by
// cb-tumblebug/cb-spider for AWS EKS). This must be rewritten to the token-broker form.
const eksKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://EXAMPLE.gr7.ap-northeast-2.eks.amazonaws.com
    certificate-authority-data: QUFB
  name: tbcluster
contexts:
- context:
    cluster: tbcluster
    user: aws-iam-user
  name: tbcluster
current-context: tbcluster
users:
- name: aws-iam-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      interactiveMode: Never
      command: aws-iam-authenticator
      args:
      - token
      - -i
      - tbcluster
`

// AKS-style kubeconfig: self-contained embedded client cert/key (as returned by
// cb-tumblebug/cb-spider for Azure AKS). This must be left untouched.
const aksKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.azmk8s.io:443
    certificate-authority-data: QUFB
  name: tbcluster
contexts:
- context:
    cluster: tbcluster
    user: clusterAdmin
  name: tbcluster
current-context: tbcluster
users:
- name: clusterAdmin
  user:
    client-certificate-data: QkJC
    client-key-data: Q0ND
    token: sometoken
`

func TestRewriteExecToBroker_EKS(t *testing.T) {
	cfg, err := clientcmd.Load([]byte(eksKubeconfig))
	if err != nil {
		t.Fatalf("load eks kubeconfig: %v", err)
	}

	if !rewriteExecToBroker(cfg, "http://tb:1323", "testns01", "ekstest01", "default", "default") {
		t.Fatal("expected exec-plugin kubeconfig to be rewritten, got false")
	}

	auth := cfg.AuthInfos["aws-iam-user"]
	if auth == nil || auth.Exec == nil {
		t.Fatal("expected rewritten user to keep an exec stanza")
	}
	if auth.Exec.Command != "sh" {
		t.Fatalf("expected broker command 'sh', got %q", auth.Exec.Command)
	}
	if len(auth.Exec.Args) != 2 || auth.Exec.Args[0] != "-c" {
		t.Fatalf("unexpected broker args: %v", auth.Exec.Args)
	}
	shellCmd := auth.Exec.Args[1]
	for _, want := range []string{
		"curl -fsS",
		"-u 'default:default'",
		"http://tb:1323/tumblebug/ns/testns01/k8sCluster/ekstest01/token",
		"jq -ce .execCredential",
	} {
		if !strings.Contains(shellCmd, want) {
			t.Errorf("broker shell command missing %q\n got: %s", want, shellCmd)
		}
	}

	// The rewritten config must still serialize and reload cleanly, preserving cluster info.
	out, err := clientcmd.Write(*cfg)
	if err != nil {
		t.Fatalf("write rewritten kubeconfig: %v", err)
	}
	reloaded, err := clientcmd.Load(out)
	if err != nil {
		t.Fatalf("reload rewritten kubeconfig: %v", err)
	}
	if got := reloaded.Clusters["tbcluster"].Server; got != "https://EXAMPLE.gr7.ap-northeast-2.eks.amazonaws.com" {
		t.Errorf("cluster server not preserved: %q", got)
	}
}

func TestRewriteExecToBroker_AKS_NoRewrite(t *testing.T) {
	cfg, err := clientcmd.Load([]byte(aksKubeconfig))
	if err != nil {
		t.Fatalf("load aks kubeconfig: %v", err)
	}

	if rewriteExecToBroker(cfg, "http://tb:1323", "testns01", "k8stest01", "default", "default") {
		t.Fatal("self-contained kubeconfig must not be rewritten")
	}
	if cfg.AuthInfos["clusterAdmin"].Token != "sometoken" {
		t.Error("self-contained credentials must be left untouched")
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"default":   `'default'`,
		"a:b":       `'a:b'`,
		"pa'ss":     `'pa'\''ss'`,
		"qwe1212!Q": `'qwe1212!Q'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
