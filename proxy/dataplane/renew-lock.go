package dataplane

import "net/http"

// swagger:route POST /queues/{id}/messages/{msg_id}/renew-lock queues renewLock
func RenewLock(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"lock": "renewed"})
}
