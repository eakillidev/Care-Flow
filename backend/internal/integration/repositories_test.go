package integration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/eakillidev/Care-Flow/backend/internal/visits"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserRepository(t *testing.T) {
	pool := setupTestDatabase(t)
	repository := users.NewPostgresRepository(pool)
	ctx := context.Background()

	user := &users.User{
		FirstName:    "Taylor",
		LastName:     "Caregiver",
		Email:        "taylor@example.test",
		PasswordHash: "test-placeholder-hash",
		Role:         users.RoleCaregiver,
	}
	if err := repository.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Fatal("expected generated user ID")
	}

	byID, err := repository.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user by ID: %v", err)
	}
	if byID.Email != user.Email {
		t.Fatalf("expected email %q, got %q", user.Email, byID.Email)
	}

	byEmail, err := repository.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if byEmail.ID != user.ID {
		t.Fatalf("expected ID %s, got %s", user.ID, byEmail.ID)
	}

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 listed user, got %d", len(listed))
	}

	duplicate := &users.User{
		FirstName:    "Another",
		LastName:     "User",
		Email:        "  TAYLOR@EXAMPLE.TEST  ",
		PasswordHash: "test-placeholder-hash",
		Role:         users.RoleCoordinator,
	}
	if err := repository.Create(ctx, duplicate); !errors.Is(err, database.ErrConflict) {
		t.Fatalf("expected duplicate email conflict, got %v", err)
	}
}

func TestPatientRepository(t *testing.T) {
	pool := setupTestDatabase(t)
	repository := patients.NewPostgresRepository(pool)
	ctx := context.Background()

	patient := &patients.Patient{
		FirstName: "Morgan",
		LastName:  "Patient",
		Address:   "10 Initial Street",
		Latitude:  40.7128,
		Longitude: -74.0060,
	}
	if err := repository.Create(ctx, patient); err != nil {
		t.Fatalf("create patient: %v", err)
	}

	stored, err := repository.GetByID(ctx, patient.ID)
	if err != nil {
		t.Fatalf("get patient: %v", err)
	}
	if stored.Address != patient.Address {
		t.Fatalf("expected address %q, got %q", patient.Address, stored.Address)
	}

	patient.Address = "20 Updated Avenue"
	patient.Latitude = 40.7306
	patient.Longitude = -73.9352
	if err := repository.Update(ctx, patient); err != nil {
		t.Fatalf("update patient: %v", err)
	}

	updated, err := repository.GetByID(ctx, patient.ID)
	if err != nil {
		t.Fatalf("get updated patient: %v", err)
	}
	if updated.Address != patient.Address || updated.Latitude != patient.Latitude {
		t.Fatalf("patient update was not persisted: %#v", updated)
	}

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list patients: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 listed patient, got %d", len(listed))
	}
}

func TestVisitRepository(t *testing.T) {
	pool := setupTestDatabase(t)
	ctx := context.Background()

	userRepository := users.NewPostgresRepository(pool)
	patientRepository := patients.NewPostgresRepository(pool)
	visitRepository := visits.NewPostgresRepository(pool)

	caregiver := &users.User{
		FirstName:    "Robin",
		LastName:     "Caregiver",
		Email:        "robin@example.test",
		PasswordHash: "test-placeholder-hash",
		Role:         users.RoleCaregiver,
	}
	if err := userRepository.Create(ctx, caregiver); err != nil {
		t.Fatalf("create caregiver: %v", err)
	}
	patient := &patients.Patient{
		FirstName: "Jamie",
		LastName:  "Patient",
		Address:   "30 Visit Road",
		Latitude:  39.9526,
		Longitude: -75.1652,
	}
	if err := patientRepository.Create(ctx, patient); err != nil {
		t.Fatalf("create patient: %v", err)
	}

	start := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	visit := &visits.Visit{
		PatientID:      patient.ID,
		CaregiverID:    caregiver.ID,
		ScheduledStart: start,
		ScheduledEnd:   start.Add(time.Hour),
		Status:         visits.StatusScheduled,
		EVVStatus:      visits.EVVStatusPending,
	}
	if err := visitRepository.Create(ctx, visit); err != nil {
		t.Fatalf("create visit: %v", err)
	}

	stored, err := visitRepository.GetByID(ctx, visit.ID)
	if err != nil {
		t.Fatalf("get visit: %v", err)
	}
	if stored.PatientID != patient.ID || stored.CaregiverID != caregiver.ID {
		t.Fatalf("unexpected stored visit: %#v", stored)
	}

	invalidForeignKey := &visits.Visit{
		PatientID:      uuid.New(),
		CaregiverID:    caregiver.ID,
		ScheduledStart: start,
		ScheduledEnd:   start.Add(time.Hour),
		Status:         visits.StatusScheduled,
		EVVStatus:      visits.EVVStatusPending,
	}
	if err := visitRepository.Create(ctx, invalidForeignKey); !errors.Is(err, database.ErrForeignKeyViolation) {
		t.Fatalf("expected foreign-key violation, got %v", err)
	}

	invalidSchedule := &visits.Visit{
		PatientID:      patient.ID,
		CaregiverID:    caregiver.ID,
		ScheduledStart: start,
		ScheduledEnd:   start.Add(-time.Minute),
		Status:         visits.StatusScheduled,
		EVVStatus:      visits.EVVStatusPending,
	}
	if err := visitRepository.Create(ctx, invalidSchedule); !errors.Is(err, database.ErrConstraintViolation) {
		t.Fatalf("expected schedule constraint violation, got %v", err)
	}

	caregiverVisits, err := visitRepository.ListByCaregiver(ctx, caregiver.ID)
	if err != nil {
		t.Fatalf("list visits by caregiver: %v", err)
	}
	if len(caregiverVisits) != 1 || caregiverVisits[0].ID != visit.ID {
		t.Fatalf("unexpected caregiver visits: %#v", caregiverVisits)
	}

	allVisits, err := visitRepository.List(ctx)
	if err != nil {
		t.Fatalf("list visits: %v", err)
	}
	if len(allVisits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(allVisits))
	}

	updated, err := visitRepository.UpdateStatus(ctx, visit.ID, visits.StatusInProgress)
	if err != nil {
		t.Fatalf("update visit status: %v", err)
	}
	if updated.Status != visits.StatusInProgress {
		t.Fatalf("expected in-progress status, got %q", updated.Status)
	}

	checkIn := start.Add(2 * time.Minute)
	updated, err = visitRepository.UpdateCheckIn(ctx, visit.ID, checkIn, 39.9527, -75.1651)
	if err != nil {
		t.Fatalf("update check-in: %v", err)
	}
	if updated.ActualCheckIn == nil || !updated.ActualCheckIn.Equal(checkIn) || updated.CheckInLatitude == nil {
		t.Fatalf("check-in was not persisted: %#v", updated)
	}

	checkOut := start.Add(58 * time.Minute)
	updated, err = visitRepository.UpdateCheckOut(ctx, visit.ID, checkOut, 39.9528, -75.1650)
	if err != nil {
		t.Fatalf("update check-out: %v", err)
	}
	if updated.ActualCheckOut == nil || !updated.ActualCheckOut.Equal(checkOut) || updated.CheckOutLongitude == nil {
		t.Fatalf("check-out was not persisted: %#v", updated)
	}

	exception := "development verification exception"
	updated, err = visitRepository.UpdateEVVResult(ctx, visit.ID, visits.EVVStatusException, &exception)
	if err != nil {
		t.Fatalf("update EVV result: %v", err)
	}
	if updated.EVVStatus != visits.EVVStatusException || updated.EVVException == nil || *updated.EVVException != exception {
		t.Fatalf("EVV result was not persisted: %#v", updated)
	}
}

func setupTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create admin pool: %v", err)
	}
	if err := adminPool.Ping(ctx); err != nil {
		adminPool.Close()
		t.Fatalf("connect to integration database: %v", err)
	}

	schema := "careflow_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		adminPool.Close()
		t.Fatalf("create test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		adminPool.Close()
		t.Fatalf("parse test database URL: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		adminPool.Close()
		t.Fatalf("create test pool: %v", err)
	}

	migrationPath, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		pool.Close()
		adminPool.Close()
		t.Fatalf("resolve migrations path: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrationPath, "up"); err != nil {
		pool.Close()
		adminPool.Close()
		t.Fatalf("apply test migrations: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := adminPool.Exec(cleanupContext, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		adminPool.Close()
	})

	return pool
}
