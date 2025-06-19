package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type PrometheusClient struct {
	logger *logrus.Logger
}

func NewPrometheusClient(logger *logrus.Logger, metricsAddress string, metrics *PrometheusMetrics) *PrometheusClient {
	var reg = prometheus.NewRegistry()
	logger.Info("Starting Prometheus metrics server on ", metricsAddress)
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		version.NewCollector("example"),
		metrics.ActiveConnections,
		metrics.RequestCounter,
		metrics.CpuTemp,
		metrics.HdFailures,
		metrics.RequestDurations,
	)
	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	// go http.ListenAndServe(server.metricsAddress, nil)

	return &PrometheusClient{
		logger: logger,
	}
}
