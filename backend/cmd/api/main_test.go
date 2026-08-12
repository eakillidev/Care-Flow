package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/google/uuid"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router := testRouter(t, fakeDatabase{})
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	expected := `{"database":"ok","service":"careflow-api","status":"ok"}`
	if strings.TrimSpace(response.Body.String()) != expected {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}

func TestHealthWhenDatabaseIsUnavailable(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	router := testRouter(t, fakeDatabase{err: errors.New("database unavailable")})
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}

	expected := `{"database":"unavailable","service":"careflow-api","status":"unhealthy"}`
	if strings.TrimSpace(response.Body.String()) != expected {
		t.Fatalf("unexpected response body: %s", response.Body.String())
	}
}

type fakeDatabase struct {
	err error
}

func (database fakeDatabase) Ping(context.Context) error {
	return database.err
}

type fakeUsers struct{}

func (fakeUsers) Create(context.Context, *users.User) error { return nil }
func (fakeUsers) GetByID(context.Context, uuid.UUID) (*users.User, error) {
	return nil, errors.New("unused")
}
func (fakeUsers) GetByEmail(context.Context, string) (*users.User, error) {
	return nil, errors.New("unused")
}
func (fakeUsers) List(context.Context) ([]users.User, error)                   { return nil, nil }
func (fakeUsers) ListByRole(context.Context, users.Role) ([]users.User, error) { return nil, nil }

func testRouter(t *testing.T, database databasePinger) http.Handler {
	t.Helper()
	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	repository := fakeUsers{}
	handler := auth.NewHandler(auth.NewService(repository, tokens), repository)
	return newRouter(database, handler, nil, nil, nil, tokens)
}
