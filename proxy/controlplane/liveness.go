package controlplane

import (
	"net/http"
)

// swagger:route GET /healthz telemetry liveness
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
