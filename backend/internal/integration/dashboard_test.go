package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/eakillidev/Care-Flow/backend/internal/caregivers"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/eakillidev/Care-Flow/backend/internal/visits"
	"github.com/google/uuid"
)

func TestCoordinatorVisitDashboard(t *testing.T) {
	pool := setupTestDatabase(t)
	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)
	coordinator := createSchedulingUser(t, userRepository, "Dashboard Coordinator", users.RoleCoordinator)
	caregiverOne := createSchedulingUser(t, userRepository, "Dashboard Caregiver", users.RoleCaregiver)
	caregiverTwo := createSchedulingUser(t, userRepository, "Second Dashboard", users.RoleCaregiver)
	patientOne := &patients.Patient{FirstName: "First", LastName: "Patient", Address: "One", Latitude: 39.2904, Longitude: -76.6122}
	patientTwo := &patients.Patient{FirstName: "Second", LastName: "Patient", Address: "Two", Latitude: 40.0, Longitude: -75.0}
	for _, patient := range []*patients.Patient{patientOne, patientTwo} {
		if err := patientRepository.Create(context.Background(), patient); err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}
	day := time.Date(2026, time.August, 10, 9, 0, 0, 0, time.UTC)
	createDashboardVisit(t, visitRepository, patientOne.ID, caregiverOne.ID, day, visits.StatusScheduled, visits.EVVStatusPending, nil)
	verified := createDashboardVisit(t, visitRepository, patientOne.ID, caregiverOne.ID, day.AddDate(0, 0, 1), visits.StatusCompleted, visits.EVVStatusVerified, nil)
	reason := "late_check_in,outside_geofence"
	exception := createDashboardVisit(t, visitRepository, patientTwo.ID, caregiverTwo.ID, day.AddDate(0, 0, 2), visits.StatusCompleted, visits.EVVStatusException, &reason)
	createDashboardVisit(t, visitRepository, patientTwo.ID, caregiverTwo.ID, day.AddDate(0, 0, 3), visits.StatusCancelled, visits.EVVStatusPending, nil)

	tokens, _ := auth.NewTokenManager("dashboard-secret", time.Hour)
	visitHandler := visits.NewHandler(visits.NewService(visitRepository, patientRepository, userRepository))
	router := schedulingRouter(tokens, patients.NewHandler(patients.NewService(patientRepository)), caregivers.NewHandler(userRepository), visitHandler)
	coordinatorToken := issueSchedulingToken(t, tokens, coordinator)
	caregiverToken := issueSchedulingToken(t, tokens, caregiverOne)

	checks := []struct {
		query string
		want  int
	}{
		{"", 4},
		{"?status=completed", 2},
		{"?evv_status=exception", 1},
		{"?caregiver_id=" + caregiverOne.ID.String(), 2},
		{"?patient_id=" + patientTwo.ID.String(), 2},
		{"?from=2026-08-11&to=2026-08-12", 2},
		{"?status=completed&evv_status=exception&caregiver_id=" + caregiverTwo.ID.String() + "&patient_id=" + patientTwo.ID.String() + "&from=2026-08-12&to=2026-08-12", 1},
	}
	for _, check := range checks {
		items := requestJSON[[]visits.Detail](t, router, http.MethodGet, "/api/visits"+check.query, nil, coordinatorToken, http.StatusOK)
		if len(items) != check.want {
			t.Fatalf("query %q: expected %d visits, got %d", check.query, check.want, len(items))
		}
	}
	for _, query := range []string{"?status=bad", "?evv_status=bad", "?caregiver_id=bad", "?patient_id=bad", "?from=bad", "?from=2026-08-12&to=2026-08-10"} {
		assertSchedulingStatus(t, router, http.MethodGet, "/api/visits"+query, nil, coordinatorToken, http.StatusBadRequest)
	}

	summary := requestJSON[visits.Summary](t, router, http.MethodGet, "/api/visits/evv-summary", nil, coordinatorToken, http.StatusOK)
	if summary.TotalVisits != 4 || summary.Scheduled != 1 || summary.Completed != 2 || summary.Cancelled != 1 || summary.EVVVerified != 1 || summary.EVVExceptions != 1 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	ranged := requestJSON[visits.Summary](t, router, http.MethodGet, "/api/visits/evv-summary?from=2026-08-11&to=2026-08-12", nil, coordinatorToken, http.StatusOK)
	if ranged.TotalVisits != 2 || ranged.Completed != 2 {
		t.Fatalf("unexpected ranged summary: %#v", ranged)
	}
	assertSchedulingStatus(t, router, http.MethodGet, "/api/visits/evv-summary", nil, caregiverToken, http.StatusForbidden)
	assertSchedulingStatus(t, router, http.MethodGet, "/api/visits", nil, caregiverToken, http.StatusForbidden)

	verifiedDetail := requestJSON[visits.Detail](t, router, http.MethodGet, "/api/visits/"+verified.ID.String(), nil, coordinatorToken, http.StatusOK)
	if verifiedDetail.EVV.Status != visits.EVVStatusVerified || len(verifiedDetail.EVV.ExceptionReasons) != 0 || verifiedDetail.EVV.CheckIn.DistanceFromPatientMeters == nil {
		t.Fatalf("unexpected verified detail: %#v", verifiedDetail.EVV)
	}
	exceptionDetail := requestJSON[visits.Detail](t, router, http.MethodGet, "/api/visits/"+exception.ID.String(), nil, coordinatorToken, http.StatusOK)
	if len(exceptionDetail.EVV.ExceptionReasons) != 2 || exceptionDetail.EVV.CheckOut.DistanceFromPatientMeters == nil || *exceptionDetail.EVV.CheckOut.DistanceFromPatientMeters < 1000 {
		t.Fatalf("unexpected exception detail: %#v", exceptionDetail.EVV)
	}
}

func createDashboardVisit(t *testing.T, repository visits.Repository, patientID, caregiverID uuid.UUID, start time.Time, status visits.Status, evvStatus visits.EVVStatus, exception *string) *visits.Visit {
	t.Helper()
	checkIn := start.Add(2 * time.Minute)
	checkOut := start.Add(58 * time.Minute)
	nearLat, nearLon := 39.2905, -76.6121
	farLat, farLon := 40.01, -75.0
	visit := &visits.Visit{PatientID: patientID, CaregiverID: caregiverID, ScheduledStart: start, ScheduledEnd: start.Add(time.Hour), Status: status, EVVStatus: evvStatus, EVVException: exception}
	if status == visits.StatusCompleted {
		visit.ActualCheckIn, visit.ActualCheckOut = &checkIn, &checkOut
		visit.CheckInLatitude, visit.CheckInLongitude = &nearLat, &nearLon
		if evvStatus == visits.EVVStatusException {
			visit.CheckOutLatitude, visit.CheckOutLongitude = &farLat, &farLon
		} else {
			visit.CheckOutLatitude, visit.CheckOutLongitude = &nearLat, &nearLon
		}
	}
	if err := repository.Create(context.Background(), visit); err != nil {
		t.Fatalf("create dashboard visit: %v", err)
	}
	return visit
}
