package dataplane

import "net/http"

// swagger:route DELETE /queues/{id}/deadletter/{msg_id} queues deleteDLQMessage
func DeleteDLQMessage(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"deadletter": "deleted"})
}
