package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kokaq/server/proxy/services"
	"github.com/kokaq/server/proxy/types"
)

// swagger:route PUT /queues/{id} queues updateQueue
func UpdateQueue(w http.ResponseWriter, r *http.Request) {
	q := services.NewQueueServiceFromContext(r.Context())
	id := chi.URLParam(r, "id")
	var req types.QueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	queue := services.Queue{Name: req.Name, Description: req.Description, MaxSize: req.MaxSize}
	respondJSON(w, http.StatusOK, q.UpdateQueue(id, queue))
}
