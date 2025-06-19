package controlplane

import "net/http"

// swagger:route POST /queues/{id}/purge queues purgeQueue
func PurgeQueue(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"purged": "true"})
}
