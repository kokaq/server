package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type OIDCConfig struct {
	Issuer   string
	ClientID string
}

func NewOIDCMiddleware(config OIDCConfig) func(http.Handler) http.Handler {
	provider, err := oidc.NewProvider(context.Background(), config.Issuer)
	if err != nil {
		panic("failed to connect to OIDC provider: " + err.Error())
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or malformed Authorization header", http.StatusUnauthorized)
				return
			}

			idTokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			idToken, err := verifier.Verify(r.Context(), idTokenStr)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Optional: extract and pass claims to context
			var claims map[string]interface{}
			if err := idToken.Claims(&claims); err != nil {
				http.Error(w, "Failed to parse token claims", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), "user", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
