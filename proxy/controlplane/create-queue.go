package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/kokaq/server/proxy/services"
	"github.com/kokaq/server/proxy/types"
)

// swagger:route POST /queues queues createQueue
func CreateQueue(w http.ResponseWriter, r *http.Request) {
	q := services.NewQueueServiceFromContext(r.Context())
	var req types.QueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	queue := services.Queue{Name: req.Name, Description: req.Description, MaxSize: req.MaxSize}
	respondJSON(w, http.StatusCreated, q.CreateQueue(queue))
}
