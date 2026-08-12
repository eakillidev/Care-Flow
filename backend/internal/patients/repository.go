package patients

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, patient *Patient) error
	GetByID(ctx context.Context, id uuid.UUID) (*Patient, error)
	List(ctx context.Context) ([]Patient, error)
	Update(ctx context.Context, patient *Patient) error
}
