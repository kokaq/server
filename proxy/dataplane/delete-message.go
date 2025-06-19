package dataplane

import "net/http"

// swagger:route DELETE /queues/{id}/messages/{msg_id} queues deleteMessage
func DeleteMessage(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
