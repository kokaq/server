package internals

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/kokaq/core/internals/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// Telemetry event names as constants
const (
	EventAuthFailedNoMetadata    = "auth_failed_no_metadata"
	EventAuthFailedInvalidCreds  = "auth_failed_invalid_credentials"
	EventServerListenFailed      = "server_listen_failed"
	EventServerStarted           = "server_started"
	EventServerStoppedWithError  = "server_stopped_with_error"
	EventServerStopped           = "server_stopped"
	EventServerStopping          = "server_stopping"
	EventServerStoppedGracefully = "server_stopped_gracefully"
	EventServerStopTimeout       = "server_stop_timeout"
	EventServerCleanup           = "server_cleanup"
	EventHealthCheckRequested    = "health_check_requested"
	EventHealthCheckResponded    = "health_check_responded"
	EventRequestTimeout          = "request_timeout"
	EventAuthFailedInvalidOIDC   = "auth_failed_invalid_oidc"
)

type KokaqServer struct {
	grpcServer      *grpc.Server
	listener        net.Listener
	cleanup         func()
	telemetryLogger TelemetryLogger
	healthServer    *health.Server
	requestTimeout  time.Duration
}

func NewKokaqServer(cleanup func(), telemetryLogger TelemetryLogger, requestTimeout time.Duration) (*KokaqServer, error) {
	var unaryInterceptors []grpc.UnaryServerInterceptor
	// if requestTimeout > 0 {
	// 	unaryInterceptors = append(unaryInterceptors, requestTimeoutUnaryInterceptor(requestTimeout, telemetryLogger))
	// }
	opts := []grpc.ServerOption{}
	if len(unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}
	grpcSrv := grpc.NewServer(opts...)
	// pb.RegisterQueueDataServer(grpcSrv, )
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, &TelemetryHealthServer{
		HealthServer:    healthSrv,
		telemetryLogger: telemetryLogger,
	})
	return &KokaqServer{
		grpcServer:      grpcSrv,
		cleanup:         cleanup,
		telemetryLogger: telemetryLogger,
		healthServer:    healthSrv,
		requestTimeout:  requestTimeout,
	}, nil
}

func (s *KokaqServer) Start(address string, register func(server *grpc.Server)) error {
	var err error

	register(s.grpcServer)

	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		logger.ConsoleLog("Error", "failed to listen: %v", err)
		if s.telemetryLogger != nil {
			s.telemetryLogger.LogEvent(EventServerListenFailed, map[string]interface{}{
				"error":   err.Error(),
				"address": address,
			})
		}
		return err
	}
	logger.ConsoleLog("INFO", "gRPC server listening without TLS on %s", address)
	if s.telemetryLogger != nil {
		s.telemetryLogger.LogEvent(EventServerStarted, map[string]interface{}{
			"address": address,
		})
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if serveErr := s.grpcServer.Serve(s.listener); serveErr != nil {
			logger.ConsoleLog("ERROR", "gRPC server stopped with error: %v", serveErr)
			if s.telemetryLogger != nil {
				s.telemetryLogger.LogEvent(EventServerStoppedWithError, map[string]interface{}{
					"error": serveErr.Error(),
				})
			}
		} else {
			if s.telemetryLogger != nil {
				s.telemetryLogger.LogEvent(EventServerStopped, nil)
			}
		}
	}()
	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	// Set health status to SERVING
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	}
	return nil
}

func (s *KokaqServer) Stop(ctx context.Context) error {
	logger.ConsoleLog("INFO", "stopping gRPC server")
	if s.telemetryLogger != nil {
		s.telemetryLogger.LogEvent(EventServerStopping, nil)
	}
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		logger.ConsoleLog("WARN", "gRPC server stopped gracefully")
		if s.telemetryLogger != nil {
			s.telemetryLogger.LogEvent(EventServerStoppedGracefully, nil)
		}
	case <-ctx.Done():
		logger.ConsoleLog("ERROR", "timeout waiting for server to stop: %v", ctx.Err())
		if s.telemetryLogger != nil {
			s.telemetryLogger.LogEvent(EventServerStopTimeout, map[string]interface{}{
				"error": ctx.Err().Error(),
			})
		}
		return ctx.Err()
	}
	// Do cleanup
	if s.cleanup != nil {
		logger.ConsoleLog("INFO", "performing cleanup before shutdown")
		if s.telemetryLogger != nil {
			s.telemetryLogger.LogEvent(EventServerCleanup, nil)
		}
		s.cleanup()
	}
	return nil
}
