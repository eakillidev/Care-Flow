package seed

import (
	"context"
	"fmt"

	"github.com/eakillidev/Care-Flow/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DevelopmentPassword = "careflow123"

func Development(ctx context.Context, pool *pgxpool.Pool) error {
	passwordHash, err := auth.HashPassword(DevelopmentPassword)
	if err != nil {
		return fmt.Errorf("hash development password: %w", err)
	}

	transaction, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, `
        INSERT INTO users (id, first_name, last_name, email, password_hash, role)
        VALUES
            ('10000000-0000-4000-8000-000000000001', 'Alex', 'Morgan', 'coordinator@careflow.local', $1, 'coordinator'),
            ('10000000-0000-4000-8000-000000000002', 'Carmen', 'Caregiver', 'caregiver1@careflow.local', $1, 'caregiver'),
            ('10000000-0000-4000-8000-000000000003', 'Drew', 'Caregiver', 'caregiver2@careflow.local', $1, 'caregiver')
        ON CONFLICT (id) DO UPDATE SET
            first_name = EXCLUDED.first_name,
            last_name = EXCLUDED.last_name,
            email = EXCLUDED.email,
            password_hash = EXCLUDED.password_hash,
            role = EXCLUDED.role,
            updated_at = CURRENT_TIMESTAMP`, passwordHash); err != nil {
		return fmt.Errorf("seed users: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
        INSERT INTO patients (id, first_name, last_name, address, latitude, longitude)
        VALUES
            ('20000000-0000-4000-8000-000000000001', 'Pat', 'One', '101 Example Avenue, Testville', 40.7128, -74.0060),
            ('20000000-0000-4000-8000-000000000002', 'Pat', 'Two', '202 Sample Street, Testville', 40.7306, -73.9352)
        ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed patients: %w", err)
	}

	if _, err := transaction.Exec(ctx, `
        INSERT INTO visits (
            id, patient_id, caregiver_id, scheduled_start, scheduled_end, status, evv_status
        )
        VALUES
            (
                '30000000-0000-4000-8000-000000000001',
                '20000000-0000-4000-8000-000000000001',
                '10000000-0000-4000-8000-000000000002',
                CURRENT_DATE + INTERVAL '1 day 9 hours',
                CURRENT_DATE + INTERVAL '1 day 10 hours',
                'scheduled',
                'pending'
            ),
            (
                '30000000-0000-4000-8000-000000000002',
                '20000000-0000-4000-8000-000000000002',
                '10000000-0000-4000-8000-000000000002',
                CURRENT_DATE + INTERVAL '1 day 11 hours',
                CURRENT_DATE + INTERVAL '1 day 12 hours',
                'scheduled',
                'pending'
            ),
            (
                '30000000-0000-4000-8000-000000000003',
                '20000000-0000-4000-8000-000000000001',
                '10000000-0000-4000-8000-000000000003',
                CURRENT_DATE + INTERVAL '2 days 14 hours',
                CURRENT_DATE + INTERVAL '2 days 15 hours 30 minutes',
                'scheduled',
                'pending'
            )
        ON CONFLICT (id) DO NOTHING`); err != nil {
		return fmt.Errorf("seed visits: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}
