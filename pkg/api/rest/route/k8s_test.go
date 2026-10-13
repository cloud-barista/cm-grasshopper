package route

import (
	"testing"

	"github.com/labstack/echo/v4"
)

// The migration entry point is what cm-cicada calls, so the path and method are
// a contract rather than an implementation detail.
func TestK8sRoute(t *testing.T) {
	e := echo.New()
	K8s(e)

	for _, route := range e.Routes() {
		if route.Method == "POST" && route.Path == "/grasshopper/k8s/migrate" {
			return
		}
	}

	t.Errorf("POST /grasshopper/k8s/migrate is not registered: %+v", e.Routes())
}

// cm-butterfly reads the candidate list from this path before building a /k8s/migrate request.
func TestK8sCandidatesRoute(t *testing.T) {
	e := echo.New()
	K8s(e)

	for _, route := range e.Routes() {
		if route.Method == "POST" && route.Path == "/grasshopper/k8s/candidates" {
			return
		}
	}

	t.Errorf("POST /grasshopper/k8s/candidates is not registered: %+v", e.Routes())
}
