package controlplane

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kokaq/server/proxy/services"
)

// swagger:route DELETE /queues/{id} queues deleteQueue
func DeleteQueue(w http.ResponseWriter, r *http.Request) {
	q := services.NewQueueServiceFromContext(r.Context())
	id := chi.URLParam(r, "id")
	q.DeleteQueue(id)
	w.WriteHeader(http.StatusNoContent)
}
