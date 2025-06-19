package controlplane

import "net/http"

// swagger:route GET /quotas quotas getQuotas
func GetQuotas(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"quotas": "usage"})
}
