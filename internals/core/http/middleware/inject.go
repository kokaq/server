package middleware

import (
	"context"
	"net/http"
)

func InjectService[T any](name string, s func() (T, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			svc, e := s()
			if e != nil {
				http.Error(w, "Injection Failed", http.StatusInternalServerError)
			}
			ctx := context.WithValue(r.Context(), name, svc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
