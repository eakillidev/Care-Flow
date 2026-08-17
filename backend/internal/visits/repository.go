package visits

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, visit *Visit) error
	GetByID(ctx context.Context, id uuid.UUID) (*Visit, error)
	List(ctx context.Context) ([]Visit, error)
	ListByCaregiver(ctx context.Context, caregiverID uuid.UUID) ([]Visit, error)
	GetDetail(ctx context.Context, id uuid.UUID) (*Detail, error)
	GetDetailForCaregiver(ctx context.Context, id, caregiverID uuid.UUID) (*Detail, error)
	ListDetails(ctx context.Context) ([]Detail, error)
	ListDetailsFiltered(ctx context.Context, filter Filter) ([]Detail, error)
	Summary(ctx context.Context, filter Filter) (Summary, error)
	ListDetailsByCaregiver(ctx context.Context, caregiverID uuid.UUID) ([]Detail, error)
	HasOverlap(ctx context.Context, caregiverID uuid.UUID, start, end time.Time, excludeID *uuid.UUID) (bool, error)
	UpdateSchedule(ctx context.Context, id, caregiverID uuid.UUID, start, end time.Time) (*Visit, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Visit, error)
	CheckIn(ctx context.Context, id, caregiverID uuid.UUID, checkedInAt time.Time, latitude, longitude float64, evvStatus EVVStatus, exception *string) (*Visit, error)
	CheckOut(ctx context.Context, id, caregiverID uuid.UUID, checkedOutAt time.Time, latitude, longitude float64, evvStatus EVVStatus, exception *string) (*Visit, error)
	UpdateCheckIn(ctx context.Context, id uuid.UUID, checkedInAt time.Time, latitude, longitude float64) (*Visit, error)
	UpdateCheckOut(ctx context.Context, id uuid.UUID, checkedOutAt time.Time, latitude, longitude float64) (*Visit, error)
	UpdateEVVResult(ctx context.Context, id uuid.UUID, status EVVStatus, exception *string) (*Visit, error)
}
