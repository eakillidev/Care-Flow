package integration_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/evv"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/eakillidev/Care-Flow/backend/internal/visits"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestEVVHTTPWorkflow(t *testing.T) {
	pool := setupTestDatabase(t)
	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)
	caregiver := createSchedulingUser(t, userRepository, "EVV Caregiver", users.RoleCaregiver)
	otherCaregiver := createSchedulingUser(t, userRepository, "Other Caregiver", users.RoleCaregiver)
	coordinator := createSchedulingUser(t, userRepository, "EVV Coordinator", users.RoleCoordinator)
	patient := &patients.Patient{
		FirstName: "EVV", LastName: "Patient", Address: "100 Main Street",
		Latitude: 39.2904, Longitude: -76.6122,
	}
	if err := patientRepository.Create(context.Background(), patient); err != nil {
		t.Fatalf("create EVV patient: %v", err)
	}

	fixedNow := time.Date(2026, time.August, 15, 9, 4, 12, 0, time.UTC)
	service := visits.NewServiceWithEVVClock(
		visitRepository, patientRepository, userRepository,
		evv.NewService(200, 15*time.Minute),
		func() time.Time { return fixedNow },
	)
	handler := visits.NewHandler(service)
	tokens, err := auth.NewTokenManager("evv-integration-secret", time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	caregiverToken := issueSchedulingToken(t, tokens, caregiver)
	otherToken := issueSchedulingToken(t, tokens, otherCaregiver)
	coordinatorToken := issueSchedulingToken(t, tokens, coordinator)
	router := evvRouter(tokens, handler)

	validVisit := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow.Add(-4*time.Minute), visits.StatusScheduled)
	verified := requestJSON[visits.EVVResponse](t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-in",
		map[string]any{"latitude": 39.2905, "longitude": -76.6121}, caregiverToken, http.StatusOK)
	if verified.Status != visits.StatusInProgress || verified.ActualCheckIn == nil ||
		verified.EVV.Status != evv.StatusVerified || len(verified.EVV.Exceptions) != 0 {
		t.Fatalf("unexpected verified check-in: %#v", verified)
	}
	stored, err := visitRepository.GetByID(context.Background(), validVisit.ID)
	if err != nil {
		t.Fatalf("get checked-in visit: %v", err)
	}
	if stored.CheckInLatitude == nil || stored.CheckInLongitude == nil || stored.Status != visits.StatusInProgress {
		t.Fatalf("check-in values were not stored: %#v", stored)
	}
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-in",
		map[string]any{"latitude": 39.2905, "longitude": -76.6121}, caregiverToken, http.StatusConflict)
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-in",
		map[string]any{"latitude": 39.2905, "longitude": -76.6121}, otherToken, http.StatusNotFound)
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-in",
		map[string]any{"latitude": 39.2905, "longitude": -76.6121}, coordinatorToken, http.StatusForbidden)

	checkedOut := requestJSON[visits.EVVResponse](t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-out",
		map[string]any{"latitude": 39.2904, "longitude": -76.6122}, caregiverToken, http.StatusOK)
	if checkedOut.Status != visits.StatusCompleted || checkedOut.ActualCheckOut == nil || checkedOut.EVV.Status != evv.StatusVerified {
		t.Fatalf("unexpected verified checkout: %#v", checkedOut)
	}
	stored, err = visitRepository.GetByID(context.Background(), validVisit.ID)
	if err != nil || stored.CheckOutLatitude == nil || stored.Status != visits.StatusCompleted {
		t.Fatalf("checkout values were not stored: visit=%#v err=%v", stored, err)
	}
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+validVisit.ID.String()+"/check-out",
		map[string]any{"latitude": 39.2904, "longitude": -76.6122}, caregiverToken, http.StatusConflict)

	earlyOutside := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow.Add(16*time.Minute), visits.StatusScheduled)
	exceptionResult := requestJSON[visits.EVVResponse](t, router, http.MethodPost,
		"/api/caregiver/visits/"+earlyOutside.ID.String()+"/check-in",
		map[string]any{"latitude": 39.3004, "longitude": -76.6122}, caregiverToken, http.StatusOK)
	if exceptionResult.EVV.Status != evv.StatusException || len(exceptionResult.EVV.Exceptions) != 2 {
		t.Fatalf("expected multiple EVV exceptions, got %#v", exceptionResult)
	}
	checkoutException := requestJSON[visits.EVVResponse](t, router, http.MethodPost,
		"/api/caregiver/visits/"+earlyOutside.ID.String()+"/check-out",
		map[string]any{"latitude": 39.3004, "longitude": -76.6122}, caregiverToken, http.StatusOK)
	if checkoutException.EVV.Status != evv.StatusException || len(checkoutException.EVV.Exceptions) != 3 {
		t.Fatalf("expected preserved and checkout exception, got %#v", checkoutException)
	}

	late := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow.Add(-16*time.Minute), visits.StatusScheduled)
	lateResult := requestJSON[visits.EVVResponse](t, router, http.MethodPost,
		"/api/caregiver/visits/"+late.ID.String()+"/check-in",
		map[string]any{"latitude": 39.2904, "longitude": -76.6122}, caregiverToken, http.StatusOK)
	if len(lateResult.EVV.Exceptions) != 1 || lateResult.EVV.Exceptions[0] != evv.ReasonLateCheckIn {
		t.Fatalf("expected late check-in exception, got %#v", lateResult)
	}

	cancelled := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow, visits.StatusCancelled)
	completed := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow, visits.StatusCompleted)
	scheduled := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, fixedNow, visits.StatusScheduled)
	for _, visit := range []*visits.Visit{cancelled, completed} {
		assertSchedulingStatus(t, router, http.MethodPost,
			"/api/caregiver/visits/"+visit.ID.String()+"/check-in",
			map[string]any{"latitude": 39.2904, "longitude": -76.6122}, caregiverToken, http.StatusConflict)
	}
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+scheduled.ID.String()+"/check-out",
		map[string]any{"latitude": 39.2904, "longitude": -76.6122}, caregiverToken, http.StatusConflict)
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+scheduled.ID.String()+"/check-in",
		map[string]any{"latitude": 91, "longitude": -76.6122}, caregiverToken, http.StatusBadRequest)
	assertSchedulingStatus(t, router, http.MethodPost,
		"/api/caregiver/visits/"+earlyOutside.ID.String()+"/check-out",
		map[string]any{"latitude": 39.2904, "longitude": -76.6122}, otherToken, http.StatusNotFound)
}

func TestConcurrentCheckInOnlyOneSucceeds(t *testing.T) {
	pool := setupTestDatabase(t)
	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)
	caregiver := createSchedulingUser(t, userRepository, "Concurrent Caregiver", users.RoleCaregiver)
	patient := &patients.Patient{FirstName: "Concurrent", LastName: "Patient", Address: "Test", Latitude: 39.2904, Longitude: -76.6122}
	if err := patientRepository.Create(context.Background(), patient); err != nil {
		t.Fatalf("create patient: %v", err)
	}
	now := time.Date(2026, time.August, 15, 9, 0, 0, 0, time.UTC)
	visit := createEVVVisit(t, visitRepository, patient.ID, caregiver.ID, now, visits.StatusScheduled)
	service := visits.NewServiceWithEVVClock(
		visitRepository, patientRepository, userRepository,
		evv.NewService(200, 15*time.Minute), func() time.Time { return now },
	)

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := service.CheckIn(context.Background(), visit.ID, caregiver.ID, 39.2904, -76.6122)
			errorsChannel <- err
		}()
	}
	close(start)
	waitGroup.Wait()
	close(errorsChannel)

	successes := 0
	failures := 0
	for err := range errorsChannel {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected one successful and one rejected check-in, got successes=%d failures=%d", successes, failures)
	}
}

func evvRouter(tokens *auth.TokenManager, handler *visits.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(auth.Authenticate(tokens))
	router.Group(func(caregiver chi.Router) {
		caregiver.Use(auth.RequireRole(users.RoleCaregiver))
		caregiver.Post("/api/caregiver/visits/{id}/check-in", handler.CheckIn)
		caregiver.Post("/api/caregiver/visits/{id}/check-out", handler.CheckOut)
	})
	return router
}

func createEVVVisit(
	t *testing.T,
	repository visits.Repository,
	patientID, caregiverID uuid.UUID,
	scheduledStart time.Time,
	status visits.Status,
) *visits.Visit {
	t.Helper()
	visit := &visits.Visit{
		PatientID: patientID, CaregiverID: caregiverID,
		ScheduledStart: scheduledStart, ScheduledEnd: scheduledStart.Add(time.Hour),
		Status: status, EVVStatus: visits.EVVStatusPending,
	}
	if status == visits.StatusCompleted {
		checkedInAt := scheduledStart
		checkedOutAt := scheduledStart.Add(time.Hour)
		visit.ActualCheckIn = &checkedInAt
		visit.ActualCheckOut = &checkedOutAt
	}
	if err := repository.Create(context.Background(), visit); err != nil {
		t.Fatalf("create %s EVV visit: %v", status, err)
	}
	return visit
}
