package proxy

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/kokaq/server/internals/core/http"
	"github.com/kokaq/server/internals/core/http/middleware"
	"github.com/kokaq/server/proxy/controlplane"
	"github.com/kokaq/server/proxy/dataplane"
	"github.com/kokaq/server/proxy/services"
	"github.com/sirupsen/logrus"
)

type ProxyServiceConfig struct {
	Logger           *logrus.Logger
	HttpServerConfig http.KokaqHttpServerConfig
}

type ProxyService struct {
	logger *logrus.Logger
	server *http.KokaqHttpServer
}

func NewProxyService(config ProxyServiceConfig) *ProxyService {

	var server = http.NewKokaqHttpServer(config.HttpServerConfig,
		func(r chi.Router) {
			r.Use(middleware.InjectService("queue-service",
				func() (*services.QueueService, error) {
					return services.NewQueueService(config.Logger, services.QueueServiceConfig{
						MessageContentEtcdEndpoints:    "",
						MessageContentDbEtcdTimeout:    10,
						PodMappingDbEtcdEndpoints:      "",
						PodMappingDbEtcdTimeout:        10,
						StorageServiceTcpTimeout:       10,
						StorageServicePodsFetchTimeout: 10,
						StorageServiceTcpRetries:       3,
					})
				}))
			// Quotas
			r.Get("/quotas", controlplane.GetQuotas)
			r.Post("/quotas", controlplane.SetQuotas)

			// Health
			r.Get("/healthz", controlplane.LivenessHandler)
			r.Get("/readyz", controlplane.ReadinessHandler)

			// Control Plane
			r.Route("/queues", func(r chi.Router) {
				r.Get("/", controlplane.ListQueues)
				r.Post("/", controlplane.CreateQueue)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", controlplane.GetQueue)
					r.Put("/", controlplane.UpdateQueue)
					r.Delete("/", controlplane.DeleteQueue)
					r.Get("/stats", controlplane.GetQueueStats)
					r.Post("/purge", controlplane.PurgeQueue)
					r.Route("/permissions", func(r chi.Router) {
						r.Get("/", controlplane.ListPermissions)
						r.Post("/", controlplane.SetPermissions)
					})
					r.Route("/messages", func(r chi.Router) {
						r.Post("/", dataplane.EnqueueMessage)
						r.Get("/", dataplane.ReceiveMessages)
						r.Post("/batch", dataplane.BatchEnqueue)
						r.Post("/{msg_id}/renew-lock", dataplane.RenewLock)
						r.Delete("/{msg_id}", dataplane.DeleteMessage)
					})
					r.Route("/deadletter", func(r chi.Router) {
						r.Get("/", dataplane.ListDLQ)
						r.Post("/{msg_id}", dataplane.MoveToDLQ)
						r.Delete("/{msg_id}", dataplane.DeleteDLQMessage)
					})
					r.Post("/replay", dataplane.ReplayMessages)
				})
			})

		},
	)

	return &ProxyService{
		logger: config.Logger,
		server: server,
	}
}

func (ps *ProxyService) Start(ctx context.Context) error {
	return ps.server.Start(ctx)
}
