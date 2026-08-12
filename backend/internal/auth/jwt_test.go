package auth

import (
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/google/uuid"
)

func TestTokenIssueAndValidation(t *testing.T) {
	manager := mustTokenManager(t, "test-signing-secret", time.Hour)
	userID := uuid.New()

	encoded, err := manager.Issue(userID, users.RoleCoordinator)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	identity, err := manager.Validate(encoded)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if identity.UserID != userID || identity.Role != users.RoleCoordinator {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}

func TestTokenValidationRejectsInvalidSignature(t *testing.T) {
	issuer := mustTokenManager(t, "issuer-secret", time.Hour)
	validator := mustTokenManager(t, "different-secret", time.Hour)
	encoded, err := issuer.Issue(uuid.New(), users.RoleCaregiver)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if _, err := validator.Validate(encoded); err != ErrInvalidToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestTokenValidationRejectsExpiredToken(t *testing.T) {
	manager := mustTokenManager(t, "test-signing-secret", time.Minute)
	issuedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return issuedAt }
	encoded, err := manager.Issue(uuid.New(), users.RoleCaregiver)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	manager.now = func() time.Time { return issuedAt.Add(2 * time.Minute) }
	if _, err := manager.Validate(encoded); err != ErrInvalidToken {
		t.Fatalf("expected expired token rejection, got %v", err)
	}
}

func TestTokenValidationRejectsMalformedToken(t *testing.T) {
	manager := mustTokenManager(t, "test-signing-secret", time.Hour)
	if _, err := manager.Validate("not-a-jwt"); err != ErrInvalidToken {
		t.Fatalf("expected malformed token rejection, got %v", err)
	}
}

func mustTokenManager(t *testing.T, secret string, expiration time.Duration) *TokenManager {
	t.Helper()
	manager, err := NewTokenManager(secret, expiration)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	return manager
}
