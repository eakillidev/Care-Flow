package users

import (
	"context"
	"fmt"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const userColumns = `id, first_name, last_name, email, password_hash, role, created_at, updated_at`

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Create(ctx context.Context, user *User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	user.Email = NormalizeEmail(user.Email)

	err := repository.pool.QueryRow(ctx, `
        INSERT INTO users (id, first_name, last_name, email, password_hash, role)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING created_at, updated_at`,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Email,
		user.PasswordHash,
		user.Role,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", database.NormalizeError(err))
	}
	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := repository.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", id)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", database.NormalizeError(err))
	}
	return user, nil
}

func (repository *PostgresRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row := repository.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE email = $1", NormalizeEmail(email))
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", database.NormalizeError(err))
	}
	return user, nil
}

func (repository *PostgresRepository) List(ctx context.Context) ([]User, error) {
	return repository.list(ctx, "SELECT "+userColumns+" FROM users ORDER BY created_at, id")
}

func (repository *PostgresRepository) ListByRole(ctx context.Context, role Role) ([]User, error) {
	return repository.list(
		ctx,
		"SELECT "+userColumns+" FROM users WHERE role = $1 ORDER BY last_name, first_name, id",
		role,
	)
}

func (repository *PostgresRepository) list(ctx context.Context, statement string, arguments ...any) ([]User, error) {
	rows, err := repository.pool.Query(ctx, statement, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	result := make([]User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		result = append(result, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanUser(row rowScanner) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

var _ rowScanner = pgx.Row(nil)
