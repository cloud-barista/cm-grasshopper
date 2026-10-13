package honeybee

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

func sampleKubernetes() *softwaremodel.Kubernetes {
	return &softwaremodel.Kubernetes{
		Version: "v1.30.2",
		Resources: &softwaremodel.KubernetesResources{
			Namespaces: []string{"shop", "auth", "empty"},
			Workloads: map[string]map[string]int{
				"shop":   {"deployments": 3, "services": 2},
				"auth":   {"deployments": 1, "secrets": 4},
				"batch":  {"jobs": 2},
				"empty2": {},
			},
			ClusterScopedWorkloads: map[string]int{"clusterroles": 40},
			StorageClasses: []softwaremodel.KubernetesStorageClass{
				{Name: "local-path", Provisioner: "rancher.io/local-path"},
			},
			PersistentVolumeClaims: []softwaremodel.KubernetesPersistentVolumeClaim{
				{Namespace: "shop", Name: "db-data", StorageClass: "local-path", AccessModes: []string{"ReadWriteOnce"}},
				{Namespace: "shop", Name: "cache", StorageClass: "local-path"},
				{Namespace: "auth", Name: "keys"},
			},
			HelmReleases: []softwaremodel.KubernetesHelmRelease{
				{Namespace: "shop", Name: "redis", Chart: "redis", ChartVersion: "19.0.1"},
			},
		},
	}
}

func TestBuildCandidatesMergesNamespaces(t *testing.T) {
	res := BuildCandidates(sampleKubernetes())

	var names []string
	for _, ns := range res.Namespaces {
		names = append(names, ns.Name)
	}
	want := []string{"auth", "batch", "empty", "empty2", "shop"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("namespaces = %v, want %v", names, want)
	}
	if res.Version != "v1.30.2" {
		t.Errorf("version = %q", res.Version)
	}

	for _, ns := range res.Namespaces {
		if ns.Workloads == nil || ns.PersistentVolumeClaims == nil || ns.HelmReleases == nil {
			t.Errorf("namespace %s has nil fields: %+v", ns.Name, ns)
		}
		if ns.Name == "batch" && ns.Workloads["jobs"] != 2 {
			t.Errorf("batch workloads = %v", ns.Workloads)
		}
	}
}

func TestBuildCandidatesResourceKindsSortedUnion(t *testing.T) {
	res := BuildCandidates(sampleKubernetes())

	want := []string{"deployments", "jobs", "secrets", "services"}
	if !reflect.DeepEqual(res.ResourceKinds, want) {
		t.Errorf("resourceKinds = %v, want %v", res.ResourceKinds, want)
	}
	if res.ClusterScopedWorkloads["clusterroles"] != 40 {
		t.Errorf("clusterScopedWorkloads = %v", res.ClusterScopedWorkloads)
	}
	if len(res.StorageClasses) != 1 || res.StorageClasses[0].Provisioner != "rancher.io/local-path" {
		t.Errorf("storageClasses = %+v", res.StorageClasses)
	}
}

func TestBuildCandidatesGroupsClaimsAndReleases(t *testing.T) {
	res := BuildCandidates(sampleKubernetes())

	byName := map[string]int{}
	for i, ns := range res.Namespaces {
		byName[ns.Name] = i
	}

	shop := res.Namespaces[byName["shop"]]
	if len(shop.PersistentVolumeClaims) != 2 || shop.PersistentVolumeClaims[0].Name != "db-data" ||
		!reflect.DeepEqual(shop.PersistentVolumeClaims[0].AccessModes, []string{"ReadWriteOnce"}) {
		t.Errorf("shop claims = %+v", shop.PersistentVolumeClaims)
	}
	if len(shop.HelmReleases) != 1 || shop.HelmReleases[0].ChartVersion != "19.0.1" {
		t.Errorf("shop releases = %+v", shop.HelmReleases)
	}

	auth := res.Namespaces[byName["auth"]]
	if len(auth.PersistentVolumeClaims) != 1 || auth.PersistentVolumeClaims[0].Name != "keys" {
		t.Errorf("auth claims = %+v", auth.PersistentVolumeClaims)
	}
	if len(auth.HelmReleases) != 0 {
		t.Errorf("auth releases = %+v", auth.HelmReleases)
	}
}

func TestBuildCandidatesEmptyInventory(t *testing.T) {
	for _, k8s := range []*softwaremodel.Kubernetes{
		nil,
		{Version: "v1.30.2"},
		{Version: "v1.30.2", Resources: &softwaremodel.KubernetesResources{}},
	} {
		res := BuildCandidates(k8s)
		if res.Namespaces == nil || res.ResourceKinds == nil || res.StorageClasses == nil || res.ClusterScopedWorkloads == nil {
			t.Errorf("nil fields for %+v: %+v", k8s, res)
		}
		if len(res.Namespaces) != 0 || len(res.ResourceKinds) != 0 {
			t.Errorf("non-empty result for %+v: %+v", k8s, res)
		}
	}
}

func TestKubernetesFromRefined(t *testing.T) {
	const withK8s = `{"sourceSoftwareModel":{"source_group_id":"sg-01","connection_info_list":[
		{"connection_id":"conn-cp","softwares":{"binaries":[],"packages":[],"containers":[],
		 "kubernetes":[{"version":"v1.30.2","kube_config":"","resources":{"namespaces":["shop"]}}]}}]}}`
	k8s, err := kubernetesFromRefined([]byte(withK8s), "conn-cp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k8s.Version != "v1.30.2" || !reflect.DeepEqual(k8s.Resources.Namespaces, []string{"shop"}) {
		t.Errorf("kubernetes = %+v", k8s)
	}

	for name, body := range map[string]string{
		"vm connection": `{"sourceSoftwareModel":{"source_group_id":"sg-01","connection_info_list":[
			{"connection_id":"conn-vm","softwares":{"binaries":[],"packages":[],"containers":[],"kubernetes":null}}]}}`,
		"nil resources": `{"sourceSoftwareModel":{"source_group_id":"sg-01","connection_info_list":[
			{"connection_id":"conn-vm","softwares":{"kubernetes":[{"version":"v1.30.2","kube_config":""}]}}]}}`,
		"other connection": `{"sourceSoftwareModel":{"source_group_id":"sg-01","connection_info_list":[
			{"connection_id":"conn-other","softwares":{"kubernetes":[{"version":"v1.30.2","resources":{}}]}}]}}`,
	} {
		if _, err := kubernetesFromRefined([]byte(body), "conn-vm"); !errors.Is(err, ErrNoKubernetesInventory) {
			t.Errorf("%s: err = %v, want ErrNoKubernetesInventory", name, err)
		}
	}

	if _, err := kubernetesFromRefined([]byte("not json"), "conn-vm"); err == nil || errors.Is(err, ErrNoKubernetesInventory) {
		t.Errorf("malformed body: err = %v", err)
	}
}

func withHoneybee(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/honeybee/source_group/sg-01/connection_info/conn-cp/software/refined" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}

	saved := config.CMGrasshopperConfig
	t.Cleanup(func() { config.CMGrasshopperConfig = saved })
	config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress = host
	config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort = port
}

func TestFetchCandidatesHoneybee4xxIsNoInventory(t *testing.T) {
	withHoneybee(t, http.StatusBadRequest, `{"error":"Failed to get information of the software. (ConnectionID = conn-cp)"}`)

	_, err := FetchCandidates("sg-01", "conn-cp")
	if !errors.Is(err, ErrNoKubernetesInventory) {
		t.Fatalf("err = %v, want ErrNoKubernetesInventory", err)
	}
	want := "no kubernetes inventory for this connection (honeybee: Failed to get information of the software. (ConnectionID = conn-cp))"
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
}

func TestFetchCandidatesHoneybee404IsNoInventory(t *testing.T) {
	withHoneybee(t, http.StatusNotFound, "not found")

	_, err := FetchCandidates("sg-01", "conn-cp")
	if !errors.Is(err, ErrNoKubernetesInventory) {
		t.Fatalf("err = %v, want ErrNoKubernetesInventory", err)
	}
	if want := "no kubernetes inventory for this connection (honeybee: not found)"; err.Error() != want {
		t.Errorf("err = %q, want %q", err.Error(), want)
	}
}

func TestFetchCandidatesOtherHoneybee4xxIsInternal(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		withHoneybee(t, status, `{"error":"denied"}`)

		_, err := FetchCandidates("sg-01", "conn-cp")
		if err == nil || errors.Is(err, ErrNoKubernetesInventory) {
			t.Errorf("status %d: err = %v, want an internal error", status, err)
		}
	}
}

func TestFetchCandidatesHoneybee5xxIsInternal(t *testing.T) {
	withHoneybee(t, http.StatusInternalServerError, `{"error":"boom"}`)

	_, err := FetchCandidates("sg-01", "conn-cp")
	if err == nil || errors.Is(err, ErrNoKubernetesInventory) {
		t.Fatalf("err = %v, want an internal error", err)
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("err = %q, want the upstream status", err.Error())
	}
}

func TestFetchCandidatesConnectionFailureIsInternal(t *testing.T) {
	withHoneybee(t, http.StatusOK, "")
	config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort = "1"

	_, err := FetchCandidates("sg-01", "conn-cp")
	if err == nil || errors.Is(err, ErrNoKubernetesInventory) {
		t.Fatalf("err = %v, want an internal error", err)
	}
}

func TestFetchCandidatesOK(t *testing.T) {
	withHoneybee(t, http.StatusOK, `{"sourceSoftwareModel":{"source_group_id":"sg-01","connection_info_list":[
		{"connection_id":"conn-cp","softwares":{"kubernetes":[{"version":"v1.30.2","resources":{"namespaces":["shop"]}}]}}]}}`)

	res, err := FetchCandidates("sg-01", "conn-cp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Version != "v1.30.2" || len(res.Namespaces) != 1 || res.Namespaces[0].Name != "shop" {
		t.Errorf("res = %+v", res)
	}
}
