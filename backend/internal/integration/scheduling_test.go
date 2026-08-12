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
	"github.com/eakillidev/Care-Flow/backend/internal/caregivers"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/eakillidev/Care-Flow/backend/internal/visits"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestSchedulingHTTPWorkflow(t *testing.T) {
	pool := setupTestDatabase(t)
	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)

	coordinator := createSchedulingUser(t, userRepository, "Coordinator", users.RoleCoordinator)
	caregiverOne := createSchedulingUser(t, userRepository, "Caregiver One", users.RoleCaregiver)
	caregiverTwo := createSchedulingUser(t, userRepository, "Caregiver Two", users.RoleCaregiver)

	tokens, err := auth.NewTokenManager("scheduling-test-secret", time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	coordinatorToken := issueSchedulingToken(t, tokens, coordinator)
	caregiverOneToken := issueSchedulingToken(t, tokens, caregiverOne)
	caregiverTwoToken := issueSchedulingToken(t, tokens, caregiverTwo)

	patientHandler := patients.NewHandler(patients.NewService(patientRepository))
	caregiverHandler := caregivers.NewHandler(userRepository)
	visitHandler := visits.NewHandler(visits.NewService(visitRepository, patientRepository, userRepository))
	router := schedulingRouter(tokens, patientHandler, caregiverHandler, visitHandler)

	patientPayload := map[string]any{
		"first_name": " Jane ",
		"last_name":  " Smith ",
		"address":    " 100 Main Street, Baltimore, MD ",
		"latitude":   39.2904,
		"longitude":  -76.6122,
	}
	createdPatient := requestJSON[patients.Patient](t, router, http.MethodPost, "/api/patients", patientPayload, coordinatorToken, http.StatusCreated)
	if createdPatient.FirstName != "Jane" || createdPatient.Address != "100 Main Street, Baltimore, MD" {
		t.Fatalf("patient text was not trimmed: %#v", createdPatient)
	}
	assertSchedulingStatus(t, router, http.MethodPost, "/api/patients", patientPayload, caregiverOneToken, http.StatusForbidden)
	assertSchedulingStatus(t, router, http.MethodPost, "/api/patients", map[string]any{
		"first_name": "Bad", "last_name": "Coordinates", "address": "Somewhere", "latitude": 91, "longitude": 0,
	}, coordinatorToken, http.StatusBadRequest)

	listedPatients := requestJSON[[]patients.Patient](t, router, http.MethodGet, "/api/patients", nil, coordinatorToken, http.StatusOK)
	if len(listedPatients) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(listedPatients))
	}
	retrievedPatient := requestJSON[patients.Patient](t, router, http.MethodGet, "/api/patients/"+createdPatient.ID.String(), nil, coordinatorToken, http.StatusOK)
	if retrievedPatient.ID != createdPatient.ID {
		t.Fatalf("retrieved wrong patient: %s", retrievedPatient.ID)
	}
	patientPayload["address"] = "200 Updated Avenue"
	updatedPatient := requestJSON[patients.Patient](t, router, http.MethodPut, "/api/patients/"+createdPatient.ID.String(), patientPayload, coordinatorToken, http.StatusOK)
	if updatedPatient.Address != "200 Updated Avenue" {
		t.Fatalf("patient update was not persisted: %#v", updatedPatient)
	}
	assertSchedulingStatus(t, router, http.MethodGet, "/api/patients/"+uuid.NewString(), nil, coordinatorToken, http.StatusNotFound)
	assertSchedulingStatus(t, router, http.MethodGet, "/api/patients", nil, caregiverOneToken, http.StatusForbidden)

	caregiverList := requestJSON[[]map[string]any](t, router, http.MethodGet, "/api/caregivers", nil, coordinatorToken, http.StatusOK)
	if len(caregiverList) != 2 {
		t.Fatalf("expected only 2 caregivers, got %d", len(caregiverList))
	}
	for _, item := range caregiverList {
		if item["role"] != string(users.RoleCaregiver) {
			t.Fatalf("non-caregiver returned: %#v", item)
		}
		if _, found := item["password_hash"]; found {
			t.Fatal("caregiver list exposed password hash")
		}
	}
	assertSchedulingStatus(t, router, http.MethodGet, "/api/caregivers", nil, caregiverOneToken, http.StatusForbidden)

	day := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	baseVisit := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(9*time.Hour), day.Add(12*time.Hour), http.StatusCreated)
	if baseVisit.Status != visits.StatusScheduled || baseVisit.EVVStatus != visits.EVVStatusPending || baseVisit.ActualCheckIn != nil {
		t.Fatalf("unexpected visit defaults: %#v", baseVisit)
	}
	assertSchedulingStatus(t, router, http.MethodPost, "/api/visits", visitPayload(createdPatient.ID, caregiverOne.ID, day.Add(14*time.Hour), day.Add(15*time.Hour)), caregiverOneToken, http.StatusForbidden)
	createVisitHTTP(t, router, coordinatorToken, uuid.New(), caregiverOne.ID, day.Add(14*time.Hour), day.Add(15*time.Hour), http.StatusNotFound)
	createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, uuid.New(), day.Add(14*time.Hour), day.Add(15*time.Hour), http.StatusNotFound)
	createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, coordinator.ID, day.Add(14*time.Hour), day.Add(15*time.Hour), http.StatusBadRequest)
	createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(15*time.Hour), day.Add(14*time.Hour), http.StatusBadRequest)

	createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(11*time.Hour), day.Add(13*time.Hour), http.StatusConflict)
	afterBoundary := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(12*time.Hour), day.Add(14*time.Hour), http.StatusCreated)
	beforeBoundary := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(7*time.Hour), day.Add(9*time.Hour), http.StatusCreated)
	_ = beforeBoundary

	cancelledBase := requestJSON[visits.Detail](t, router, http.MethodPost, "/api/visits/"+baseVisit.ID.String()+"/cancel", nil, coordinatorToken, http.StatusOK)
	if cancelledBase.Status != visits.StatusCancelled {
		t.Fatalf("expected cancelled visit, got %q", cancelledBase.Status)
	}
	requestJSON[visits.Detail](t, router, http.MethodPost, "/api/visits/"+baseVisit.ID.String()+"/cancel", nil, coordinatorToken, http.StatusOK)
	candidate := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverOne.ID, day.Add(9*time.Hour), day.Add(11*time.Hour), http.StatusCreated)

	caregiverTwoConflict := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverTwo.ID, day.Add(9*time.Hour), day.Add(12*time.Hour), http.StatusCreated)
	_ = caregiverTwoConflict
	assertSchedulingStatus(t, router, http.MethodPatch, "/api/visits/"+candidate.ID.String(), map[string]any{
		"caregiver_id": caregiverTwo.ID,
	}, coordinatorToken, http.StatusConflict)
	assertSchedulingStatus(t, router, http.MethodPatch, "/api/visits/"+candidate.ID.String(), map[string]any{
		"scheduled_start": day.Add(11 * time.Hour), "scheduled_end": day.Add(13 * time.Hour),
	}, coordinatorToken, http.StatusConflict)

	rescheduled := requestJSON[visits.Detail](t, router, http.MethodPatch, "/api/visits/"+candidate.ID.String(), map[string]any{
		"scheduled_start": day.Add(15 * time.Hour), "scheduled_end": day.Add(16 * time.Hour),
	}, coordinatorToken, http.StatusOK)
	if !rescheduled.ScheduledStart.Equal(day.Add(15 * time.Hour)) {
		t.Fatalf("visit was not rescheduled: %#v", rescheduled)
	}
	reassigned := requestJSON[visits.Detail](t, router, http.MethodPatch, "/api/visits/"+candidate.ID.String(), map[string]any{
		"caregiver_id": caregiverTwo.ID,
	}, coordinatorToken, http.StatusOK)
	if reassigned.Caregiver.ID != caregiverTwo.ID {
		t.Fatalf("visit was not reassigned: %#v", reassigned)
	}

	allVisits := requestJSON[[]visits.Detail](t, router, http.MethodGet, "/api/visits", nil, coordinatorToken, http.StatusOK)
	if len(allVisits) < 5 {
		t.Fatalf("expected coordinator to see all visits, got %d", len(allVisits))
	}
	coordinatorDetail := requestJSON[visits.Detail](t, router, http.MethodGet, "/api/visits/"+candidate.ID.String(), nil, coordinatorToken, http.StatusOK)
	if coordinatorDetail.ID != candidate.ID {
		t.Fatalf("coordinator retrieved wrong visit: %s", coordinatorDetail.ID)
	}

	completed := createVisitHTTP(t, router, coordinatorToken, createdPatient.ID, caregiverTwo.ID, day.Add(17*time.Hour), day.Add(18*time.Hour), http.StatusCreated)
	if _, err := visitRepository.UpdateStatus(context.Background(), completed.ID, visits.StatusCompleted); err != nil {
		t.Fatalf("mark visit completed for test: %v", err)
	}
	assertSchedulingStatus(t, router, http.MethodPost, "/api/visits/"+completed.ID.String()+"/cancel", nil, coordinatorToken, http.StatusConflict)

	caregiverTwoVisits := requestJSON[[]visits.Detail](t, router, http.MethodGet, "/api/caregiver/visits", nil, caregiverTwoToken, http.StatusOK)
	for _, detail := range caregiverTwoVisits {
		if detail.Caregiver.ID != caregiverTwo.ID {
			t.Fatalf("caregiver list exposed another assignment: %#v", detail)
		}
	}
	owned := requestJSON[visits.Detail](t, router, http.MethodGet, "/api/caregiver/visits/"+candidate.ID.String(), nil, caregiverTwoToken, http.StatusOK)
	if owned.ID != candidate.ID {
		t.Fatalf("caregiver retrieved wrong visit: %s", owned.ID)
	}
	assertSchedulingStatus(t, router, http.MethodGet, "/api/caregiver/visits/"+afterBoundary.ID.String(), nil, caregiverTwoToken, http.StatusNotFound)
	assertSchedulingStatus(t, router, http.MethodGet, "/api/caregiver/visits/"+candidate.ID.String(), nil, caregiverOneToken, http.StatusNotFound)
	assertSchedulingStatus(t, router, http.MethodPatch, "/api/visits/"+candidate.ID.String(), map[string]any{"scheduled_end": day.Add(19 * time.Hour)}, caregiverTwoToken, http.StatusForbidden)
	assertSchedulingStatus(t, router, http.MethodPost, "/api/visits/"+candidate.ID.String()+"/cancel", nil, caregiverTwoToken, http.StatusForbidden)
}

func schedulingRouter(
	tokens *auth.TokenManager,
	patientHandler *patients.Handler,
	caregiverHandler *caregivers.Handler,
	visitHandler *visits.Handler,
) http.Handler {
	router := chi.NewRouter()
	router.Use(auth.Authenticate(tokens))
	router.Group(func(coordinator chi.Router) {
		coordinator.Use(auth.RequireRole(users.RoleCoordinator))
		coordinator.Post("/api/patients", patientHandler.Create)
		coordinator.Get("/api/patients", patientHandler.List)
		coordinator.Get("/api/patients/{id}", patientHandler.Get)
		coordinator.Put("/api/patients/{id}", patientHandler.Update)
		coordinator.Get("/api/caregivers", caregiverHandler.List)
		coordinator.Post("/api/visits", visitHandler.Create)
		coordinator.Get("/api/visits", visitHandler.List)
		coordinator.Get("/api/visits/{id}", visitHandler.Get)
		coordinator.Patch("/api/visits/{id}", visitHandler.UpdateSchedule)
		coordinator.Post("/api/visits/{id}/cancel", visitHandler.Cancel)
	})
	router.Group(func(caregiver chi.Router) {
		caregiver.Use(auth.RequireRole(users.RoleCaregiver))
		caregiver.Get("/api/caregiver/visits", visitHandler.ListForCaregiver)
		caregiver.Get("/api/caregiver/visits/{id}", visitHandler.GetForCaregiver)
	})
	return router
}

func createSchedulingUser(t *testing.T, repository users.Repository, name string, role users.Role) *users.User {
	t.Helper()
	parts := strings.SplitN(name, " ", 2)
	lastName := "User"
	if len(parts) == 2 {
		lastName = parts[1]
	}
	user := &users.User{
		FirstName: parts[0], LastName: lastName, Email: strings.ReplaceAll(strings.ToLower(name), " ", ".") + "@example.test",
		PasswordHash: "test-hash", Role: role,
	}
	if err := repository.Create(context.Background(), user); err != nil {
		t.Fatalf("create %s: %v", role, err)
	}
	return user
}

func issueSchedulingToken(t *testing.T, manager *auth.TokenManager, user *users.User) string {
	t.Helper()
	token, err := manager.Issue(user.ID, user.Role)
	if err != nil {
		t.Fatalf("issue %s token: %v", user.Role, err)
	}
	return token
}

func createVisitHTTP(
	t *testing.T,
	handler http.Handler,
	token string,
	patientID, caregiverID uuid.UUID,
	start, end time.Time,
	expectedStatus int,
) visits.Detail {
	t.Helper()
	return requestJSON[visits.Detail](t, handler, http.MethodPost, "/api/visits", visitPayload(patientID, caregiverID, start, end), token, expectedStatus)
}

func visitPayload(patientID, caregiverID uuid.UUID, start, end time.Time) map[string]any {
	return map[string]any{
		"patient_id": patientID, "caregiver_id": caregiverID, "scheduled_start": start, "scheduled_end": end,
	}
}

func requestJSON[T any](
	t *testing.T,
	handler http.Handler,
	method, path string,
	body any,
	token string,
	expectedStatus int,
) T {
	t.Helper()
	response := schedulingRequest(t, handler, method, path, body, token)
	if response.Code != expectedStatus {
		t.Fatalf("%s %s: expected %d, got %d: %s", method, path, expectedStatus, response.Code, response.Body.String())
	}
	var result T
	if expectedStatus >= 200 && expectedStatus < 300 {
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode %s %s response: %v", method, path, err)
		}
	}
	return result
}

func assertSchedulingStatus(
	t *testing.T,
	handler http.Handler,
	method, path string,
	body any,
	token string,
	expectedStatus int,
) {
	t.Helper()
	response := schedulingRequest(t, handler, method, path, body, token)
	if response.Code != expectedStatus {
		t.Fatalf("%s %s: expected %d, got %d: %s", method, path, expectedStatus, response.Code, response.Body.String())
	}
}

func schedulingRequest(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
