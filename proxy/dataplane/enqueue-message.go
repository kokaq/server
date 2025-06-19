package dataplane

import "net/http"

// swagger:route POST /queues/{id}/messages queues enqueueMessage
func EnqueueMessage(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "enqueued"})
}
