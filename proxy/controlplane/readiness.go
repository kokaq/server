package controlplane

import (
	"net/http"
)

// swagger:route GET /readyz telemetry readiness
func ReadinessHandler(w http.ResponseWriter, r *http.Request) { w.Write([]byte("READY")) }
