package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/eakillidev/Care-Flow/backend/internal/users"
)

type identityContextKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	return identity, ok
}

func Authenticate(tokens *TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			header := request.Header.Get("Authorization")
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			identity, err := tokens.Validate(parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx := context.WithValue(request.Context(), identityContextKey{}, identity)
			next.ServeHTTP(w, request.WithContext(ctx))
		})
	}
}

func RequireRole(role users.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			identity, ok := IdentityFromContext(request.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if identity.Role != role {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, request)
		})
	}
}
