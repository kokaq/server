package dataplane

import "net/http"

// swagger:route POST /queues/{id}/replay queues replayMessages
func ReplayMessages(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"replay": "started"})
}
