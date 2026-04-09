package controller

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
)

var (
	okMessageMu sync.RWMutex
	okMessage   = model.SimpleMsg{}
	isReady     atomic.Bool
)

var _ = common.ErrorResponse{}

// SetOkMessage updates the readiness status message safely from any goroutine.
func SetOkMessage(msg string) {
	okMessageMu.Lock()
	okMessage.Message = msg
	okMessageMu.Unlock()
}

// SetReady marks the API server as ready (or not) for health probes.
func SetReady(ready bool) {
	isReady.Store(ready)
}

// CheckReady func is for checking Grasshopper server health.
//
//	@ID				health-check-readyz
//	@Summary		Check Ready
//	@Description	Check Grasshopper is ready
//	@Tags           [Admin] System management
//	@Accept			json
//	@Produce		json
//	@Success		200 {object}	model.SimpleMsg			"Successfully get ready state."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to check ready state."
//	@Router		/readyz [get]
func CheckReady(c echo.Context) error {
	status := http.StatusOK
	if !isReady.Load() {
		status = http.StatusServiceUnavailable
	}

	okMessageMu.RLock()
	snapshot := okMessage
	okMessageMu.RUnlock()

	return c.JSONPretty(status, &snapshot, " ")
}
