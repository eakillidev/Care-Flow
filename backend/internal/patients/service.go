package patients

import (
	"context"
	"errors"
	"strings"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/google/uuid"
)

var (
	ErrInvalidPatient  = errors.New("invalid patient")
	ErrPatientNotFound = errors.New("patient not found")
)

type Input struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Create(ctx context.Context, input Input) (*Patient, error) {
	patient, err := patientFromInput(input)
	if err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, patient); err != nil {
		return nil, err
	}
	return patient, nil
}

func (service *Service) Get(ctx context.Context, id uuid.UUID) (*Patient, error) {
	patient, err := service.repository.GetByID(ctx, id)
	if errors.Is(err, database.ErrNotFound) {
		return nil, ErrPatientNotFound
	}
	return patient, err
}

func (service *Service) List(ctx context.Context) ([]Patient, error) {
	return service.repository.List(ctx)
}

func (service *Service) Update(ctx context.Context, id uuid.UUID, input Input) (*Patient, error) {
	patient, err := patientFromInput(input)
	if err != nil {
		return nil, err
	}
	patient.ID = id
	if err := service.repository.Update(ctx, patient); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrPatientNotFound
		}
		return nil, err
	}
	return patient, nil
}

func patientFromInput(input Input) (*Patient, error) {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Address = strings.TrimSpace(input.Address)
	if input.FirstName == "" || input.LastName == "" || input.Address == "" ||
		input.Latitude < -90 || input.Latitude > 90 || input.Longitude < -180 || input.Longitude > 180 {
		return nil, ErrInvalidPatient
	}
	return &Patient{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Address:   input.Address,
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
	}, nil
}
