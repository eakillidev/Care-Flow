package visits

import (
	"context"
	"errors"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/evv"
	"github.com/eakillidev/Care-Flow/backend/internal/patients"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/google/uuid"
)

var (
	ErrInvalidSchedule             = errors.New("scheduled end must be after scheduled start")
	ErrPatientNotFound             = errors.New("patient not found")
	ErrCaregiverNotFound           = errors.New("caregiver not found")
	ErrAssignedUserNotCaregiver    = errors.New("assigned user is not a caregiver")
	ErrOverlappingVisit            = errors.New("caregiver has an overlapping visit")
	ErrVisitNotFound               = errors.New("visit not found")
	ErrVisitNotSchedulable         = errors.New("visit cannot be rescheduled")
	ErrVisitNotCancellable         = errors.New("visit cannot be cancelled")
	ErrInvalidCoordinates          = errors.New("invalid coordinates")
	ErrVisitNotAvailableForCheckIn = errors.New("visit is not available for check-in")
	ErrVisitNotInProgress          = errors.New("visit is not in progress")
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
	evv      *evv.Service
	now      func() time.Time
}

func NewService(visitRepository Repository, patientRepository patients.Repository, userRepository users.Repository) *Service {
	return NewServiceWithEVV(visitRepository, patientRepository, userRepository, evv.NewService(200, 15*time.Minute))
}

func NewServiceWithEVV(visitRepository Repository, patientRepository patients.Repository, userRepository users.Repository, evvService *evv.Service) *Service {
	return NewServiceWithEVVClock(visitRepository, patientRepository, userRepository, evvService, time.Now)
}

func NewServiceWithEVVClock(visitRepository Repository, patientRepository patients.Repository, userRepository users.Repository, evvService *evv.Service, now func() time.Time) *Service {
	return &Service{visits: visitRepository, patients: patientRepository, users: userRepository, evv: evvService, now: now}
}

type EVVResponse struct {
	VisitID        uuid.UUID  `json:"visit_id"`
	Status         Status     `json:"status"`
	ActualCheckIn  *time.Time `json:"actual_check_in,omitempty"`
	ActualCheckOut *time.Time `json:"actual_check_out,omitempty"`
	EVV            evv.Result `json:"evv"`
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
	return service.ListFiltered(ctx, Filter{})
}

func (service *Service) ListFiltered(ctx context.Context, filter Filter) ([]Detail, error) {
	details, err := service.visits.ListDetailsFiltered(ctx, filter)
	if err != nil {
		return nil, err
	}
	for index := range details {
		enrichEVV(&details[index])
	}
	return details, nil
}

func (service *Service) Summary(ctx context.Context, filter Filter) (Summary, error) {
	return service.visits.Summary(ctx, filter)
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (*Detail, error) {
	detail, err := service.visits.GetDetail(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	if err == nil {
		enrichEVV(detail)
	}
	return detail, err
}

func enrichEVV(detail *Detail) {
	detail.EVV = EVVDetail{
		Status:           detail.EVVStatus,
		ExceptionReasons: evv.ParseExceptions(detail.EVVException),
		CheckIn:          EVVPoint{Timestamp: detail.ActualCheckIn, Latitude: detail.CheckInLatitude, Longitude: detail.CheckInLongitude},
		CheckOut:         EVVPoint{Timestamp: detail.ActualCheckOut, Latitude: detail.CheckOutLatitude, Longitude: detail.CheckOutLongitude},
	}
	if detail.CheckInLatitude != nil && detail.CheckInLongitude != nil {
		distance := evv.DistanceMeters(detail.Patient.Latitude, detail.Patient.Longitude, *detail.CheckInLatitude, *detail.CheckInLongitude)
		detail.EVV.CheckIn.DistanceFromPatientMeters = &distance
	}
	if detail.CheckOutLatitude != nil && detail.CheckOutLongitude != nil {
		distance := evv.DistanceMeters(detail.Patient.Latitude, detail.Patient.Longitude, *detail.CheckOutLatitude, *detail.CheckOutLongitude)
		detail.EVV.CheckOut.DistanceFromPatientMeters = &distance
	}
}

func (service *Service) ListForCaregiver(ctx context.Context, caregiverID uuid.UUID) ([]Detail, error) {
	details, err := service.visits.ListDetailsByCaregiver(ctx, caregiverID)
	if err != nil {
		return nil, err
	}
	for index := range details {
		enrichEVV(&details[index])
	}
	return details, nil
}

func (service *Service) GetForCaregiver(ctx context.Context, id, caregiverID uuid.UUID) (*Detail, error) {
	detail, err := service.visits.GetDetailForCaregiver(ctx, id, caregiverID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotFound
	}
	if err == nil {
		enrichEVV(detail)
	}
	return detail, err
}

func (service *Service) CheckIn(ctx context.Context, id, caregiverID uuid.UUID, latitude, longitude float64) (*EVVResponse, error) {
	if !validCoordinates(latitude, longitude) {
		return nil, ErrInvalidCoordinates
	}
	visit, err := service.visits.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) || (err == nil && visit.CaregiverID != caregiverID) {
		return nil, ErrVisitNotFound
	}
	if err != nil {
		return nil, err
	}
	if visit.Status != StatusScheduled || visit.ActualCheckIn != nil {
		return nil, ErrVisitNotAvailableForCheckIn
	}
	patient, err := service.patients.GetByID(ctx, visit.PatientID)
	if err != nil {
		return nil, err
	}
	checkedInAt := service.now().UTC()
	result := service.evv.ValidateCheckIn(patient.Latitude, patient.Longitude, latitude, longitude, visit.ScheduledStart, checkedInAt)
	updated, err := service.visits.CheckIn(ctx, id, caregiverID, checkedInAt, latitude, longitude, EVVStatus(result.Status), evv.JoinExceptions(result.Exceptions))
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotAvailableForCheckIn
	}
	if err != nil {
		return nil, err
	}
	return &EVVResponse{VisitID: updated.ID, Status: updated.Status, ActualCheckIn: updated.ActualCheckIn, EVV: result}, nil
}

func (service *Service) CheckOut(ctx context.Context, id, caregiverID uuid.UUID, latitude, longitude float64) (*EVVResponse, error) {
	if !validCoordinates(latitude, longitude) {
		return nil, ErrInvalidCoordinates
	}
	visit, err := service.visits.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) || (err == nil && visit.CaregiverID != caregiverID) {
		return nil, ErrVisitNotFound
	}
	if err != nil {
		return nil, err
	}
	if visit.Status != StatusInProgress || visit.ActualCheckIn == nil || visit.ActualCheckOut != nil {
		return nil, ErrVisitNotInProgress
	}
	patient, err := service.patients.GetByID(ctx, visit.PatientID)
	if err != nil {
		return nil, err
	}
	checkedOutAt := service.now().UTC()
	result := service.evv.ValidateCheckOut(patient.Latitude, patient.Longitude, latitude, longitude, evv.ParseExceptions(visit.EVVException))
	updated, err := service.visits.CheckOut(ctx, id, caregiverID, checkedOutAt, latitude, longitude, EVVStatus(result.Status), evv.JoinExceptions(result.Exceptions))
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrVisitNotInProgress
	}
	if err != nil {
		return nil, err
	}
	return &EVVResponse{VisitID: updated.ID, Status: updated.Status, ActualCheckIn: updated.ActualCheckIn, ActualCheckOut: updated.ActualCheckOut, EVV: result}, nil
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

func validCoordinates(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}
