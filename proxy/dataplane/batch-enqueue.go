package dataplane

import "net/http"

// swagger:route POST /queues/{id}/messages/batch queues batchEnqueue
func BatchEnqueue(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"batch": "enqueued"})
}
