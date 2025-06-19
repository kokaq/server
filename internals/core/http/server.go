package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	http_middleware "github.com/kokaq/server/internals/core/http/middleware"
	"github.com/kokaq/server/internals/prometheus"
	"github.com/kokaq/server/internals/trace"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type KokaqHttpServerConfig struct {
	Port               int    `json:"port"`
	Timeout            int    `json:"timeout"`               // Timeout in seconds for each request
	MaxConnections     int    `json:"max_connections"`       // Maximum number of concurrent connections
	UseAuditLogger     bool   `json:"use_audit_logger"`      // Whether to use an audit logger
	UseFileAuditLogger bool   `json:"use_file_audit_logger"` // Whether to use a file-based audit logger
	UseTrace           bool   `json:"use_trace"`             // Whether to enable tracing
	UsePrometheus      bool   `json:"use_prometheus"`        // Whether to enable Prometheus metrics
	UseOpenTelemetry   bool   `json:"use_open_telemetry"`    //
	UseOidc            bool   `json:"use_oidc"`              //
	MetricsAddress     string `json:"metrics_address"`       // Address for Prometheus metrics
	OidcIssuer         string `json:"oidc_issuer"`
	OidcClientId       string `json:"oidc_client_id"`
	// Add more configuration options as needed
}

type KokaqHttpServer struct {
	port           int
	maxConnections int // Maximum number of concurrent connections
	router         *chi.Mux
	httpServer     *http.Server
	logger         *logrus.Logger
	auditLogger    *logrus.Logger
	tracer         *sdktrace.TracerProvider
	idleTimeout    int
	metricsAddress string
	metrics        *prometheus.PrometheusMetrics
	promClient     *prometheus.PrometheusClient
}

func NewKokaqHttpServer(config KokaqHttpServerConfig, configureRoutes func(r chi.Router)) *KokaqHttpServer {
	server := &KokaqHttpServer{
		port:           config.Port,
		idleTimeout:    config.Timeout,
		logger:         logrus.New(),
		maxConnections: config.MaxConnections,
	}
	if config.UseTrace {
		var err error
		server.logger.Info("Tracing enabled")
		// Initialize OpenTelemetry tracing
		var tracerClose func()
		server.tracer, err, tracerClose = trace.SetupTracer()
		if err != nil {
			server.logger.WithError(err).Warn("error starting open-telemetry tracing")
		}
		defer tracerClose()
	}

	if config.UseAuditLogger {
		server.logger.Info("Audit logging enabled")
		server.auditLogger = logrus.New()
		server.auditLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		server.auditLogger.SetLevel(logrus.InfoLevel)
		if config.UseFileAuditLogger {
			server.logger.Info("File audit logging enabled")
			file, err := os.OpenFile("./audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				server.logger.WithError(err).Error("Failed to open audit log file")
				server.auditLogger.SetOutput(os.Stdout)
			} else {
				server.auditLogger.SetOutput(file)
			}
		} else {
			server.logger.Info("Using stdout for audit logging")
		}
	}

	if config.UsePrometheus {
		server.logger.Info("Prometheus metrics enabled")
		server.metricsAddress = config.MetricsAddress
		server.metrics = prometheus.NewPrometheusMetrics()
		server.promClient = prometheus.NewPrometheusClient(server.logger, server.metricsAddress, server.metrics)
	}

	server.router = chi.NewRouter()

	// Basic Middleware
	server.router.Use(middleware.RequestID)
	server.router.Use(middleware.RealIP)
	server.router.Use(middleware.Recoverer)

	if config.UseOpenTelemetry {
		var otelMiddleware func(http.Handler) http.Handler = func(next http.Handler) http.Handler {
			return otelhttp.NewHandler(next, "request")
		}
		server.router.Use(otelMiddleware)
	}

	if config.UseOidc {
		var oidcmiddleware = http_middleware.NewOIDCMiddleware(http_middleware.OIDCConfig{
			Issuer:   config.OidcIssuer, // or your Auth0/Azure AD issuer
			ClientID: config.OidcClientId,
		})
		server.router.Use(oidcmiddleware)
	}
	configureRoutes(server.router)
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", server.port),
		Handler:      server.router,
		ReadTimeout:  time.Duration(server.idleTimeout) * time.Second,
		WriteTimeout: time.Duration(server.idleTimeout) * time.Second,
		IdleTimeout:  time.Duration(server.idleTimeout) * time.Second,
	}

	return server
}

func (server *KokaqHttpServer) Start(ctx context.Context) error {
	// Start TCP server
	err := server.StartInternal(ctx)
	if err != nil {
		server.logger.Error("error in http server ", err)
		return err
	}
	return nil
}

func (server *KokaqHttpServer) StartInternal(ctx context.Context) error {
	go func() {
		if err := server.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			server.logger.WithError(err).Fatal("Server failed")
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	server.logger.Info("Shutting down http server...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return server.httpServer.Shutdown(shutdownCtx)
}
