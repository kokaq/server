package dataplane

import "net/http"

// swagger:route GET /queues/{id}/messages queues receiveMessages
func ReceiveMessages(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"messages": "received"})
}
