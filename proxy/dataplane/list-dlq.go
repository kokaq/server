package dataplane

import "net/http"

// swagger:route GET /queues/{id}/deadletter queues listDLQ
func ListDLQ(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"deadletter": "list"})
}
