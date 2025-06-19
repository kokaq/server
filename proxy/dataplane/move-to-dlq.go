package dataplane

import "net/http"

// swagger:route POST /queues/{id}/deadletter/{msg_id} queues moveToDLQ
func MoveToDLQ(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"deadletter": "moved"})
}
