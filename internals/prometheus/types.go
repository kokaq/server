package prometheus

import "github.com/prometheus/client_golang/prometheus"

type PrometheusMetrics struct {
	ActiveConnections prometheus.Gauge       `json:"active_connections"` // Gauge for active connections
	RequestCounter    prometheus.Counter     `json:"request_counter"`    // Counter for total requests
	CpuTemp           prometheus.Gauge       `json:"cpu_temp"`           // Gauge for CPU temperature
	HdFailures        *prometheus.CounterVec `json:"hd_failures"`        // Counter for hard disk failures
	RequestDurations  prometheus.Histogram   `json:"request_durations"`  // Histogram for request durations
}

func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		}),
		RequestCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "request_counter",
			Help: "Total number of requests received",
		}),
		CpuTemp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cpu_temp",
			Help: "Current CPU temperature in Celsius",
		}),
		HdFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hd_failures",
			Help: "Count of hard disk failures",
		}, []string{"disk"}),
		RequestDurations: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "request_durations_seconds",
			Help:    "Histogram of request durations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10},
		}),
	}
}
