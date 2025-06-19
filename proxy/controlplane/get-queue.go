package controlplane

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kokaq/server/proxy/services"
)

// swagger:route GET /queues/{id} queues getQueue
func GetQueue(w http.ResponseWriter, r *http.Request) {
	q := services.NewQueueServiceFromContext(r.Context())
	id := chi.URLParam(r, "id")
	respondJSON(w, http.StatusOK, q.GetQueue(id))
}
