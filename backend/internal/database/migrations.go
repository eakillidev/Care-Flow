package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

type migration struct {
	version int64
	name    string
	path    string
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, directory, direction string) error {
	if _, err := pool.Exec(ctx, migrationsTable); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	switch direction {
	case "up":
		return migrateUp(ctx, pool, directory)
	case "down":
		return migrateDown(ctx, pool, directory)
	default:
		return fmt.Errorf("unsupported migration direction %q", direction)
	}
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, directory string) error {
	migrations, err := loadMigrations(directory, "up")
	if err != nil {
		return err
	}

	for _, item := range migrations {
		var applied bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)",
			item.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}
		if err := applyMigration(ctx, pool, item, true); err != nil {
			return err
		}
	}

	return nil
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool, directory string) error {
	var version int64
	if err := pool.QueryRow(ctx,
		"SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1",
	).Scan(&version); err != nil {
		return NormalizeError(err)
	}

	migrations, err := loadMigrations(directory, "down")
	if err != nil {
		return err
	}
	for _, item := range migrations {
		if item.version == version {
			return applyMigration(ctx, pool, item, false)
		}
	}

	return fmt.Errorf("down migration for version %d not found", version)
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, item migration, up bool) error {
	contents, err := os.ReadFile(item.path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", item.name, err)
	}

	transaction, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", item.name, err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("execute migration %s: %w", item.name, err)
	}

	if up {
		_, err = transaction.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			item.version,
			item.name,
		)
	} else {
		_, err = transaction.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", item.version)
	}
	if err != nil {
		return fmt.Errorf("record migration %s: %w", item.name, err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", item.name, err)
	}
	return nil
}

func loadMigrations(directory, direction string) ([]migration, error) {
	pattern := filepath.Join(directory, "*."+direction+".sql")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("find migrations: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no %s migrations found in %s", direction, directory)
	}

	items := make([]migration, 0, len(paths))
	for _, path := range paths {
		name := filepath.Base(path)
		versionText, _, found := strings.Cut(name, "_")
		if !found {
			return nil, fmt.Errorf("invalid migration filename %q", name)
		}
		version, err := strconv.ParseInt(versionText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %q: %w", name, err)
		}
		items = append(items, migration{version: version, name: name, path: path})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].version < items[j].version })
	return items, nil
}
