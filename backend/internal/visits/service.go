package visits

import (
	"context"
	"errors"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/google/uuid"
)

var (
	ErrInvalidSchedule          = errors.New("scheduled end must be after scheduled start")
	ErrPatientNotFound          = errors.New("patient not found")
	ErrCaregiverNotFound        = errors.New("caregiver not found")
	ErrAssignedUserNotCaregiver = errors.New("assigned user is not a caregiver")
	ErrOverlappingVisit         = errors.New("caregiver has an overlapping visit")
	ErrVisitNotFound            = errors.New("visit not found")
	ErrVisitNotSchedulable      = errors.New("visit cannot be rescheduled")
	ErrVisitNotCancellable      = errors.New("visit cannot be cancelled")
)

type CreateInput struct {
	PatientID      uuid.UUID `json:"patient_id"`
	CaregiverID    uuid.UUID `json:"caregiver_id"`
	ScheduledStart time.Time `json:"scheduled_start"`
	ScheduledEnd   time.Time `json:"scheduled_end"`
}

type UpdateScheduleInput struct {
	CaregiverID    *uuid.UUID `json:"caregiver_id"`
	ScheduledStart *time.Time `json:"scheduled_start"`
	ScheduledEnd   *time.Time `json:"scheduled_end"`
}

type Service struct {
	visits   Repository
	patients patients.Repository
	users    users.Repository
}

func NewService(visitRepository Repository, patientRepository patients.Repository, userRepository users.Repository) *Service {
	return &Service{visits: visitRepository, patients: patientRepository, users: userRepository}
}

func (service *Service) Create(ctx context.Context, input CreateInput) (*Detail, error) {
	if err := validateSchedule(input.ScheduledStart, input.ScheduledEnd); err != nil {
		return nil, err
	}
	if _, err := service.patients.GetByID(ctx, input.PatientID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	if err := service.validateCaregiver(ctx, input.CaregiverID); err != nil {
		return nil, err
	}
	if err := service.ensureNoOverlap(ctx, input.CaregiverID, input.ScheduledStart, input.ScheduledEnd, nil); err != nil {
		return nil, err
	}

	visit := &Visit{
		PatientID:      input.PatientID,
		CaregiverID:    input.CaregiverID,
		ScheduledStart: input.ScheduledStart,
		ScheduledEnd:   input.ScheduledEnd,
		Status:         StatusScheduled,
		EVVStatus:      EVVStatusPending,
	}
	if err := service.visits.Create(ctx, visit); err != nil {
		return nil, err
	}
	return service.visits.GetDetail(ctx, visit.ID)
}

func (service *Service) List(ctx context.Context) ([]Detail, error) {
	return service.visits.ListDetails(ctx)
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (*Detail, error) {
	detail, err := service.visits.GetDetail(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	return detail, err
}

func (service *Service) ListForCaregiver(ctx context.Context, caregiverID uuid.UUID) ([]Detail, error) {
	return service.visits.ListDetailsByCaregiver(ctx, caregiverID)
}

func (service *Service) GetForCaregiver(ctx context.Context, id, caregiverID uuid.UUID) (*Detail, error) {
	detail, err := service.visits.GetDetailForCaregiver(ctx, id, caregiverID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	return detail, err
}

func (service *Service) UpdateSchedule(
	ctx context.Context,
	id uuid.UUID,
	input UpdateScheduleInput,
) (*Detail, error) {
	visit, err := service.visits.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	if err != nil {
		return nil, err
	}
	if visit.Status != StatusScheduled {
		return nil, ErrVisitNotSchedulable
	}
	if input.CaregiverID == nil && input.ScheduledStart == nil && input.ScheduledEnd == nil {
		return nil, ErrInvalidSchedule
	}

	caregiverID := visit.CaregiverID
	start := visit.ScheduledStart
	end := visit.ScheduledEnd
	if input.CaregiverID != nil {
		caregiverID = *input.CaregiverID
	}
	if input.ScheduledStart != nil {
		start = *input.ScheduledStart
	}
	if input.ScheduledEnd != nil {
		end = *input.ScheduledEnd
	}
	if err := validateSchedule(start, end); err != nil {
		return nil, err
	}
	if err := service.validateCaregiver(ctx, caregiverID); err != nil {
		return nil, err
	}
	if err := service.ensureNoOverlap(ctx, caregiverID, start, end, &id); err != nil {
		return nil, err
	}
	if _, err := service.visits.UpdateSchedule(ctx, id, caregiverID, start, end); err != nil {
		return nil, err
	}
	return service.visits.GetDetail(ctx, id)
}

func (service *Service) Cancel(ctx context.Context, id uuid.UUID) (*Detail, error) {
	visit, err := service.visits.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	if err != nil {
		return nil, err
	}
	if visit.Status == StatusCancelled {
		return service.visits.GetDetail(ctx, id)
	}
	if visit.Status == StatusCompleted {
		return nil, ErrVisitNotCancellable
	}
	if _, err := service.visits.UpdateStatus(ctx, id, StatusCancelled); err != nil {
		return nil, err
	}
	return service.visits.GetDetail(ctx, id)
}

func (service *Service) validateCaregiver(ctx context.Context, id uuid.UUID) error {
	user, err := service.users.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return ErrCaregiverNotFound
	}
	if err != nil {
		return err
	}
	if user.Role != users.RoleCaregiver {
		return ErrAssignedUserNotCaregiver
	}
	return nil
}

func (service *Service) ensureNoOverlap(
	ctx context.Context,
	caregiverID uuid.UUID,
	start, end time.Time,
	excludeID *uuid.UUID,
) error {
	overlaps, err := service.visits.HasOverlap(ctx, caregiverID, start, end, excludeID)
	if err != nil {
		return err
	}
	if overlaps {
		return ErrOverlappingVisit
	}
	return nil
}

func validateSchedule(start, end time.Time) error {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return ErrInvalidSchedule
	}
	return nil
}
