package controlplane

import "net/http"

// swagger:route GET /queues/{id}/permissions queues listPermissions
func ListPermissions(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"permissions": "list"})
}
