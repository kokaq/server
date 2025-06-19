package controlplane

import (
	"net/http"

	"github.com/kokaq/server/proxy/services"
)

// swagger:route GET /queues queues listQueues
// @Summary List all queues
// @Tags Queue Management
// @Produce json
// @Success 200 {array} types.Queue
// @Router /queues [get]
func ListQueues(w http.ResponseWriter, r *http.Request) {
	q := services.NewQueueServiceFromContext(r.Context())
	respondJSON(w, http.StatusOK, q.ListQueues())
}
