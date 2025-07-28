package internals

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type TelemetryLogger interface {
	LogEvent(event string, fields map[string]interface{})
}

// TelemetryHealthServer wraps the default health server to add telemetry logging.
type TelemetryHealthServer struct {
	grpc_health_v1.HealthServer
	telemetryLogger TelemetryLogger
}

func (t *TelemetryHealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if t.telemetryLogger != nil {
		t.telemetryLogger.LogEvent(EventHealthCheckRequested, map[string]interface{}{
			"service": req.Service,
		})
	}
	resp, err := t.HealthServer.Check(ctx, req)
	if t.telemetryLogger != nil {
		t.telemetryLogger.LogEvent(EventHealthCheckResponded, map[string]interface{}{
			"service": req.Service,
			"status":  resp.GetStatus().String(),
			"error":   errString(err),
		})
	}
	return resp, err
}

func (t *TelemetryHealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	if t.telemetryLogger != nil {
		t.telemetryLogger.LogEvent(EventHealthCheckRequested, map[string]interface{}{
			"service": req.Service,
			"watch":   true,
		})
	}
	err := t.HealthServer.Watch(req, stream)
	if t.telemetryLogger != nil {
		t.telemetryLogger.LogEvent(EventHealthCheckResponded, map[string]interface{}{
			"service": req.Service,
			"watch":   true,
			"error":   errString(err),
		})
	}
	return err
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
