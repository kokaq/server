package controlplane

import "net/http"

// swagger:route POST /quotas quotas setQuotas
func SetQuotas(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"quotas": "set"})
}
