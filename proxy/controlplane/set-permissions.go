package controlplane

import "net/http"

// swagger:route POST /queues/{id}/permissions queues setPermissions
func SetPermissions(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"permissions": "set"})
}
