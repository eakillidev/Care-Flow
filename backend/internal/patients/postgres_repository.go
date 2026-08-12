package patients

import (
	"context"
	"fmt"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const patientColumns = `id, first_name, last_name, address, latitude, longitude, created_at, updated_at`

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Create(ctx context.Context, patient *Patient) error {
	if patient.ID == uuid.Nil {
		patient.ID = uuid.New()
	}

	err := repository.pool.QueryRow(ctx, `
        INSERT INTO patients (id, first_name, last_name, address, latitude, longitude)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING created_at, updated_at`,
		patient.ID,
		patient.FirstName,
		patient.LastName,
		patient.Address,
		patient.Latitude,
		patient.Longitude,
	).Scan(&patient.CreatedAt, &patient.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create patient: %w", database.NormalizeError(err))
	}
	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Patient, error) {
	row := repository.pool.QueryRow(ctx, "SELECT "+patientColumns+" FROM patients WHERE id = $1", id)
	patient, err := scanPatient(row)
	if err != nil {
		return nil, fmt.Errorf("get patient by ID: %w", database.NormalizeError(err))
	}
	return patient, nil
}

func (repository *PostgresRepository) List(ctx context.Context) ([]Patient, error) {
	rows, err := repository.pool.Query(ctx, "SELECT "+patientColumns+" FROM patients ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("list patients: %w", err)
	}
	defer rows.Close()

	result := make([]Patient, 0)
	for rows.Next() {
		patient, err := scanPatient(rows)
		if err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		result = append(result, *patient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patients: %w", err)
	}
	return result, nil
}

func (repository *PostgresRepository) Update(ctx context.Context, patient *Patient) error {
	err := repository.pool.QueryRow(ctx, `
        UPDATE patients
        SET first_name = $2,
            last_name = $3,
            address = $4,
            latitude = $5,
            longitude = $6,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1
        RETURNING created_at, updated_at`,
		patient.ID,
		patient.FirstName,
		patient.LastName,
		patient.Address,
		patient.Latitude,
		patient.Longitude,
	).Scan(&patient.CreatedAt, &patient.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update patient: %w", database.NormalizeError(err))
	}
	return nil
}

func scanPatient(row interface{ Scan(...any) error }) (*Patient, error) {
	var patient Patient
	err := row.Scan(
		&patient.ID,
		&patient.FirstName,
		&patient.LastName,
		&patient.Address,
		&patient.Latitude,
		&patient.Longitude,
		&patient.CreatedAt,
		&patient.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &patient, nil
}
