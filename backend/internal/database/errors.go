package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound            = errors.New("record not found")
	ErrConflict            = errors.New("record conflicts with existing data")
	ErrForeignKeyViolation = errors.New("referenced record does not exist")
	ErrConstraintViolation = errors.New("record violates a database constraint")
)

func NormalizeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return err
	}

	switch postgresError.Code {
	case "23505":
		return ErrConflict
	case "23503":
		return ErrForeignKeyViolation
	case "23514", "23502":
		return ErrConstraintViolation
	default:
		return err
	}
}
