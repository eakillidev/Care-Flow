package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/go-chi/chi/v5"
)

func TestAuthenticationHTTPFlow(t *testing.T) {
	pool := setupTestDatabase(t)
	repository := users.NewPostgresRepository(pool)
	passwordHash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}

	coordinator := &users.User{
		FirstName:    "Alex",
		LastName:     "Coordinator",
		Email:        "coordinator@example.test",
		PasswordHash: passwordHash,
		Role:         users.RoleCoordinator,
	}
	caregiver := &users.User{
		FirstName:    "Carmen",
		LastName:     "Caregiver",
		Email:        "caregiver@example.test",
		PasswordHash: passwordHash,
		Role:         users.RoleCaregiver,
	}
	for _, user := range []*users.User{coordinator, caregiver} {
		if err := repository.Create(context.Background(), user); err != nil {
			t.Fatalf("create test user: %v", err)
		}
	}

	tokens, err := auth.NewTokenManager("integration-test-secret", time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	handler := auth.NewHandler(auth.NewService(repository, tokens), repository)
	router := chi.NewRouter()
	router.Post("/api/auth/login", handler.Login)
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Authenticate(tokens))
		protected.Get("/api/me", handler.Me)
		protected.With(auth.RequireRole(users.RoleCoordinator)).Get("/api/coordinator/ping", auth.CoordinatorPing)
		protected.With(auth.RequireRole(users.RoleCaregiver)).Get("/api/caregiver/ping", auth.CaregiverPing)
	})

	coordinatorToken := login(t, router, "  COORDINATOR@EXAMPLE.TEST ", "test-password")
	caregiverToken := login(t, router, caregiver.Email, "test-password")

	assertStatus(t, router, http.MethodPost, "/api/auth/login", `{"email":"coordinator@example.test","password":"wrong"}`, "", http.StatusUnauthorized)
	assertStatus(t, router, http.MethodPost, "/api/auth/login", `{"email":"unknown@example.test","password":"test-password"}`, "", http.StatusUnauthorized)
	assertStatus(t, router, http.MethodGet, "/api/me", "", "", http.StatusUnauthorized)
	assertStatus(t, router, http.MethodGet, "/api/coordinator/ping", "", coordinatorToken, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/coordinator/ping", "", caregiverToken, http.StatusForbidden)
	assertStatus(t, router, http.MethodGet, "/api/caregiver/ping", "", caregiverToken, http.StatusOK)
	assertStatus(t, router, http.MethodGet, "/api/caregiver/ping", "", coordinatorToken, http.StatusForbidden)

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer "+coordinatorToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected /api/me status 200, got %d: %s", response.Code, response.Body.String())
	}
	var profile map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile["id"] != coordinator.ID.String() || profile["email"] != coordinator.Email {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	for _, sensitive := range []string{"password", "password_hash"} {
		if _, found := profile[sensitive]; found {
			t.Fatalf("profile exposed %q", sensitive)
		}
	}
}

func login(t *testing.T, handler http.Handler, email, password string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("encode login request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login failed with status %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password_hash") {
		t.Fatal("login response exposed password hash")
	}
	var result auth.LoginResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if result.Token == "" {
		t.Fatal("login response did not contain a token")
	}
	return result.Token
}

func assertStatus(t *testing.T, handler http.Handler, method, path, body, token string, expected int) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != expected {
		t.Fatalf("%s %s: expected status %d, got %d: %s", method, path, expected, response.Code, response.Body.String())
	}
}
