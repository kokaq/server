package controlplane

import "net/http"

// swagger:route GET /queues/{id}/stats queues getQueueStats
func GetQueueStats(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"stats": "coming soon"})
}
