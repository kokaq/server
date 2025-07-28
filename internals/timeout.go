package internals

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func requestTimeoutUnaryInterceptor(timeout time.Duration, telemetryLogger TelemetryLogger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		done := make(chan struct{})
		var resp interface{}
		var err error
		go func() {
			resp, err = handler(ctx, req)
			close(done)
		}()
		select {
		case <-done:
			return resp, err
		case <-ctx.Done():
			if telemetryLogger != nil {
				telemetryLogger.LogEvent(EventRequestTimeout, map[string]interface{}{
					"method": info.FullMethod,
					"error":  ctx.Err().Error(),
				})
			}
			return nil, status.Error(codes.DeadlineExceeded, "request timed out")
		}
	}
}
