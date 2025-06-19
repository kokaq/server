package middleware

import (
	"net/http"
)

func TracingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Placeholder for OpenTelemetry or other tracing logic
		return next
	}
}
