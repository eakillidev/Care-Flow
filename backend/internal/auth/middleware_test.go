package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/google/uuid"
)

func TestAuthenticationMiddleware(t *testing.T) {
	manager := mustTokenManager(t, "middleware-secret", time.Hour)
	validToken, err := manager.Issue(uuid.New(), users.RoleCaregiver)
	if err != nil {
		t.Fatalf("issue valid token: %v", err)
	}
	wrongSigner := mustTokenManager(t, "wrong-secret", time.Hour)
	invalidSignature, err := wrongSigner.Issue(uuid.New(), users.RoleCaregiver)
	if err != nil {
		t.Fatalf("issue wrong-signature token: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{name: "valid token", header: "Bearer " + validToken, wantStatus: http.StatusNoContent},
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "malformed token", header: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized},
		{name: "invalid signature", header: "Bearer " + invalidSignature, wantStatus: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if _, ok := IdentityFromContext(request.Context()); !ok {
					t.Fatal("expected typed identity in context")
				}
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()

			Authenticate(manager)(next).ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
		})
	}
}

func TestAuthenticationMiddlewareRejectsExpiredToken(t *testing.T) {
	manager := mustTokenManager(t, "middleware-secret", time.Minute)
	issuedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return issuedAt }
	encoded, err := manager.Issue(uuid.New(), users.RoleCaregiver)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	manager.now = func() time.Time { return issuedAt.Add(2 * time.Minute) }

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+encoded)
	response := httptest.NewRecorder()
	Authenticate(manager)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("expired token reached protected handler")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}
