package visits

import (
	"context"
	"fmt"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const visitColumns = `
    id, patient_id, caregiver_id, scheduled_start, scheduled_end,
    actual_check_in, actual_check_out,
    check_in_latitude, check_in_longitude,
    check_out_latitude, check_out_longitude,
    status, evv_status, evv_exception, created_at, updated_at`

const detailColumns = `
    v.id,
    p.id, p.first_name, p.last_name, p.address,
    u.id, u.first_name, u.last_name,
    v.scheduled_start, v.scheduled_end,
    v.actual_check_in, v.actual_check_out,
    v.check_in_latitude, v.check_in_longitude,
    v.check_out_latitude, v.check_out_longitude,
    v.status, v.evv_status, v.evv_exception, v.created_at, v.updated_at`

const detailFrom = `
    FROM visits v
    JOIN patients p ON p.id = v.patient_id
    JOIN users u ON u.id = v.caregiver_id`

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Create(ctx context.Context, visit *Visit) error {
	if visit.ID == uuid.Nil {
		visit.ID = uuid.New()
	}

	err := repository.pool.QueryRow(ctx, `
        INSERT INTO visits (
            id, patient_id, caregiver_id, scheduled_start, scheduled_end,
            actual_check_in, actual_check_out,
            check_in_latitude, check_in_longitude,
            check_out_latitude, check_out_longitude,
            status, evv_status, evv_exception
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        RETURNING created_at, updated_at`,
		visit.ID,
		visit.PatientID,
		visit.CaregiverID,
		visit.ScheduledStart,
		visit.ScheduledEnd,
		visit.ActualCheckIn,
		visit.ActualCheckOut,
		visit.CheckInLatitude,
		visit.CheckInLongitude,
		visit.CheckOutLatitude,
		visit.CheckOutLongitude,
		visit.Status,
		visit.EVVStatus,
		visit.EVVException,
	).Scan(&visit.CreatedAt, &visit.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create visit: %w", database.NormalizeError(err))
	}
	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Visit, error) {
	return repository.getByID(ctx, id)
}

func (repository *PostgresRepository) List(ctx context.Context) ([]Visit, error) {
	return repository.list(ctx, "SELECT "+visitColumns+" FROM visits ORDER BY scheduled_start, id")
}

func (repository *PostgresRepository) ListByCaregiver(ctx context.Context, caregiverID uuid.UUID) ([]Visit, error) {
	return repository.list(
		ctx,
		"SELECT "+visitColumns+" FROM visits WHERE caregiver_id = $1 ORDER BY scheduled_start, id",
		caregiverID,
	)
}

func (repository *PostgresRepository) GetDetail(ctx context.Context, id uuid.UUID) (*Detail, error) {
	return repository.getDetail(ctx, "SELECT "+detailColumns+detailFrom+" WHERE v.id = $1", id)
}

func (repository *PostgresRepository) GetDetailForCaregiver(
	ctx context.Context,
	id, caregiverID uuid.UUID,
) (*Detail, error) {
	return repository.getDetail(
		ctx,
		"SELECT "+detailColumns+detailFrom+" WHERE v.id = $1 AND v.caregiver_id = $2",
		id,
		caregiverID,
	)
}

func (repository *PostgresRepository) ListDetails(ctx context.Context) ([]Detail, error) {
	return repository.listDetails(ctx, "SELECT "+detailColumns+detailFrom+" ORDER BY v.scheduled_start, v.id")
}

func (repository *PostgresRepository) ListDetailsByCaregiver(
	ctx context.Context,
	caregiverID uuid.UUID,
) ([]Detail, error) {
	return repository.listDetails(
		ctx,
		"SELECT "+detailColumns+detailFrom+" WHERE v.caregiver_id = $1 ORDER BY v.scheduled_start, v.id",
		caregiverID,
	)
}

func (repository *PostgresRepository) HasOverlap(
	ctx context.Context,
	caregiverID uuid.UUID,
	start, end time.Time,
	excludeID *uuid.UUID,
) (bool, error) {
	var overlaps bool
	err := repository.pool.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM visits
            WHERE caregiver_id = $1
              AND status IN ('scheduled', 'in_progress')
              AND scheduled_start < $3
              AND scheduled_end > $2
              AND ($4::uuid IS NULL OR id <> $4)
        )`, caregiverID, start, end, excludeID).Scan(&overlaps)
	if err != nil {
		return false, fmt.Errorf("check visit overlap: %w", err)
	}
	return overlaps, nil
}

func (repository *PostgresRepository) UpdateSchedule(
	ctx context.Context,
	id, caregiverID uuid.UUID,
	start, end time.Time,
) (*Visit, error) {
	return repository.updateAndGet(ctx, id, `
        UPDATE visits
        SET caregiver_id = $2,
            scheduled_start = $3,
            scheduled_end = $4,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1`, caregiverID, start, end)
}

func (repository *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) (*Visit, error) {
	return repository.updateAndGet(ctx, id, `
        UPDATE visits SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		status,
	)
}

func (repository *PostgresRepository) UpdateCheckIn(
	ctx context.Context,
	id uuid.UUID,
	checkedInAt time.Time,
	latitude, longitude float64,
) (*Visit, error) {
	return repository.updateAndGet(ctx, id, `
        UPDATE visits
        SET actual_check_in = $2,
            check_in_latitude = $3,
            check_in_longitude = $4,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1`,
		checkedInAt,
		latitude,
		longitude,
	)
}

func (repository *PostgresRepository) UpdateCheckOut(
	ctx context.Context,
	id uuid.UUID,
	checkedOutAt time.Time,
	latitude, longitude float64,
) (*Visit, error) {
	return repository.updateAndGet(ctx, id, `
        UPDATE visits
        SET actual_check_out = $2,
            check_out_latitude = $3,
            check_out_longitude = $4,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1`,
		checkedOutAt,
		latitude,
		longitude,
	)
}

func (repository *PostgresRepository) UpdateEVVResult(
	ctx context.Context,
	id uuid.UUID,
	status EVVStatus,
	exception *string,
) (*Visit, error) {
	return repository.updateAndGet(ctx, id, `
        UPDATE visits
        SET evv_status = $2,
            evv_exception = $3,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1`,
		status,
		exception,
	)
}

func (repository *PostgresRepository) updateAndGet(
	ctx context.Context,
	id uuid.UUID,
	statement string,
	arguments ...any,
) (*Visit, error) {
	arguments = append([]any{id}, arguments...)
	commandTag, err := repository.pool.Exec(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("update visit: %w", database.NormalizeError(err))
	}
	if commandTag.RowsAffected() == 0 {
		return nil, fmt.Errorf("update visit: %w", database.ErrNotFound)
	}
	return repository.getByID(ctx, id)
}

func (repository *PostgresRepository) getByID(ctx context.Context, id uuid.UUID) (*Visit, error) {
	row := repository.pool.QueryRow(ctx, "SELECT "+visitColumns+" FROM visits WHERE id = $1", id)
	visit, err := scanVisit(row)
	if err != nil {
		return nil, fmt.Errorf("get visit by ID: %w", database.NormalizeError(err))
	}
	return visit, nil
}

func (repository *PostgresRepository) list(ctx context.Context, statement string, arguments ...any) ([]Visit, error) {
	rows, err := repository.pool.Query(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list visits: %w", err)
	}
	defer rows.Close()

	result := make([]Visit, 0)
	for rows.Next() {
		visit, err := scanVisit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan visit: %w", err)
		}
		result = append(result, *visit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visits: %w", err)
	}
	return result, nil
}

func (repository *PostgresRepository) getDetail(
	ctx context.Context,
	statement string,
	arguments ...any,
) (*Detail, error) {
	detail, err := scanDetail(repository.pool.QueryRow(ctx, statement, arguments...))
	if err != nil {
		return nil, fmt.Errorf("get visit detail: %w", database.NormalizeError(err))
	}
	return detail, nil
}

func (repository *PostgresRepository) listDetails(
	ctx context.Context,
	statement string,
	arguments ...any,
) ([]Detail, error) {
	rows, err := repository.pool.Query(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list visit details: %w", err)
	}
	defer rows.Close()

	result := make([]Detail, 0)
	for rows.Next() {
		detail, err := scanDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("scan visit detail: %w", err)
		}
		result = append(result, *detail)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visit details: %w", err)
	}
	return result, nil
}

func scanVisit(row interface{ Scan(...any) error }) (*Visit, error) {
	var visit Visit
	err := row.Scan(
		&visit.ID,
		&visit.PatientID,
		&visit.CaregiverID,
		&visit.ScheduledStart,
		&visit.ScheduledEnd,
		&visit.ActualCheckIn,
		&visit.ActualCheckOut,
		&visit.CheckInLatitude,
		&visit.CheckInLongitude,
		&visit.CheckOutLatitude,
		&visit.CheckOutLongitude,
		&visit.Status,
		&visit.EVVStatus,
		&visit.EVVException,
		&visit.CreatedAt,
		&visit.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &visit, nil
}

func scanDetail(row interface{ Scan(...any) error }) (*Detail, error) {
	var detail Detail
	err := row.Scan(
		&detail.ID,
		&detail.Patient.ID,
		&detail.Patient.FirstName,
		&detail.Patient.LastName,
		&detail.Patient.Address,
		&detail.Caregiver.ID,
		&detail.Caregiver.FirstName,
		&detail.Caregiver.LastName,
		&detail.ScheduledStart,
		&detail.ScheduledEnd,
		&detail.ActualCheckIn,
		&detail.ActualCheckOut,
		&detail.CheckInLatitude,
		&detail.CheckInLongitude,
		&detail.CheckOutLatitude,
		&detail.CheckOutLongitude,
		&detail.Status,
		&detail.EVVStatus,
		&detail.EVVException,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

var _ interface{ Scan(...any) error } = pgx.Row(nil)
