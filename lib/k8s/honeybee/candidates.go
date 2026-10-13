package honeybee

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	restcommon "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	veleromodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/velero"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

// ErrNoKubernetesInventory is returned when cm-honeybee has no kubernetes entry for the
// connection, which is the case for every connection that is not a control plane node.
var ErrNoKubernetesInventory = errors.New("no kubernetes inventory for this connection")

// FetchCandidates reads the refined software inventory of a cm-honeybee connection and
// returns what can be picked for a selective kubernetes migration.
func FetchCandidates(sgID, connID string) (*veleromodel.CandidatesResponse, error) {
	sgID = strings.TrimSpace(sgID)
	connID = strings.TrimSpace(connID)
	if sgID == "" {
		return nil, errors.New("sourceGroupId is required")
	}
	if connID == "" {
		return nil, errors.New("connectionId is required")
	}

	base, err := baseURL()
	if err != nil {
		return nil, err
	}

	data, err := restcommon.GetHTTPRequest(base+"/honeybee/source_group/"+url.PathEscape(sgID)+
		"/connection_info/"+url.PathEscape(connID)+"/software/refined", "", "")
	if err != nil {
		return nil, refinedFetchError(err)
	}

	k8s, err := kubernetesFromRefined(data, connID)
	if err != nil {
		return nil, err
	}

	return BuildCandidates(k8s), nil
}

// refinedFetchError maps a failed software/refined call. cm-honeybee answers 400 when the
// source group does not exist or the connection has no collected software, which means
// there is no inventory to pick from; 404 means the same. Other statuses (401, 403, 429
// after retries, 5xx) and transport errors stay internal failures.
func refinedFetchError(err error) error {
	var statusErr *restcommon.HTTPStatusError
	if errors.As(err, &statusErr) &&
		(statusErr.StatusCode == http.StatusBadRequest || statusErr.StatusCode == http.StatusNotFound) {
		msg := strings.TrimSpace(statusErr.Body)
		var body restcommon.ErrorResponse
		if json.Unmarshal([]byte(statusErr.Body), &body) == nil && body.Error != "" {
			msg = body.Error
		}
		return fmt.Errorf("%w (honeybee: %s)", ErrNoKubernetesInventory, msg)
	}
	return fmt.Errorf("failed to fetch refined software from cm-honeybee: %w", err)
}

// kubernetesFromRefined picks the kubernetes entry of connID out of a cm-honeybee
// software/refined response, which wraps the SoftwareList in a SourceSoftwareModel.
func kubernetesFromRefined(data []byte, connID string) (*softwaremodel.Kubernetes, error) {
	var model softwaremodel.SourceSoftwareModel
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("failed to parse cm-honeybee refined software response: %w", err)
	}

	for _, conn := range model.SourceSoftwareModel.ConnectionInfoList {
		if conn.ConnectionId != connID {
			continue
		}
		if len(conn.Softwares.Kubernetes) == 0 || conn.Softwares.Kubernetes[0].Resources == nil {
			return nil, ErrNoKubernetesInventory
		}
		k8s := conn.Softwares.Kubernetes[0]
		return &k8s, nil
	}

	return nil, ErrNoKubernetesInventory
}

// BuildCandidates turns a collected kubernetes inventory into the candidate list.
// Namespaces come from the collected namespace list plus any namespace that only shows
// up in workloads, claims or helm releases, sorted by name. Slices and maps are never
// nil so the JSON response always carries every field.
func BuildCandidates(k8s *softwaremodel.Kubernetes) *veleromodel.CandidatesResponse {
	res := &veleromodel.CandidatesResponse{
		Namespaces:             []veleromodel.CandidateNamespace{},
		ResourceKinds:          []string{},
		StorageClasses:         []veleromodel.CandidateStorageClass{},
		ClusterScopedWorkloads: map[string]int{},
	}
	if k8s == nil {
		return res
	}
	res.Version = k8s.Version
	if k8s.Resources == nil {
		return res
	}
	r := k8s.Resources

	namespaces := map[string]*veleromodel.CandidateNamespace{}
	ensure := func(name string) *veleromodel.CandidateNamespace {
		ns, ok := namespaces[name]
		if !ok {
			ns = &veleromodel.CandidateNamespace{
				Name:                   name,
				Workloads:              map[string]int{},
				PersistentVolumeClaims: []veleromodel.CandidatePersistentVolumeClaim{},
				HelmReleases:           []veleromodel.CandidateHelmRelease{},
			}
			namespaces[name] = ns
		}
		return ns
	}

	for _, name := range r.Namespaces {
		ensure(name)
	}

	kinds := map[string]struct{}{}
	for name, workloads := range r.Workloads {
		ns := ensure(name)
		for kind, count := range workloads {
			ns.Workloads[kind] = count
			kinds[kind] = struct{}{}
		}
	}

	for _, pvc := range r.PersistentVolumeClaims {
		ns := ensure(pvc.Namespace)
		ns.PersistentVolumeClaims = append(ns.PersistentVolumeClaims, veleromodel.CandidatePersistentVolumeClaim{
			Name:         pvc.Name,
			StorageClass: pvc.StorageClass,
			AccessModes:  pvc.AccessModes,
		})
	}

	for _, rel := range r.HelmReleases {
		ns := ensure(rel.Namespace)
		ns.HelmReleases = append(ns.HelmReleases, veleromodel.CandidateHelmRelease{
			Name:         rel.Name,
			Chart:        rel.Chart,
			ChartVersion: rel.ChartVersion,
		})
	}

	for _, ns := range namespaces {
		res.Namespaces = append(res.Namespaces, *ns)
	}
	sort.Slice(res.Namespaces, func(i, j int) bool {
		return res.Namespaces[i].Name < res.Namespaces[j].Name
	})

	for kind := range kinds {
		res.ResourceKinds = append(res.ResourceKinds, kind)
	}
	sort.Strings(res.ResourceKinds)

	for _, sc := range r.StorageClasses {
		res.StorageClasses = append(res.StorageClasses, veleromodel.CandidateStorageClass{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
		})
	}

	for kind, count := range r.ClusterScopedWorkloads {
		res.ClusterScopedWorkloads[kind] = count
	}

	return res
}
