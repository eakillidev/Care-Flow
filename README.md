# CareFlow

CareFlow is a full-stack homecare visit management platform for caregiver scheduling, Electronic Visit Verification (EVV), and visit exception management.

## Overview

CareFlow currently provides an Angular shell, a Go API, a PostgreSQL persistence layer, and JWT authentication for coordinator and caregiver roles. Application workflows and UI features are intentionally deferred.

## Tech Stack

- Frontend: Angular 21 and TypeScript
- Backend: Go 1.24, chi, and pgx/pgxpool
- Authentication: bcrypt passwords and HMAC-SHA256 JWT access tokens
- Database: PostgreSQL 17
- Local development: Docker and Docker Compose

## Project Structure

```text
Care-Flow/
|-- frontend/                 Angular application
|-- backend/
|   |-- cmd/api/              API entry point
|   |-- cmd/migrate/          Migration command
|   |-- cmd/seed/             Development seed command
|   |-- internal/             Models and PostgreSQL repositories
|   `-- migrations/           Versioned up/down SQL migrations
|-- docker-compose.yml
|-- .env.example
`-- README.md
```

## Database Schema

- `users`: caregivers and coordinators; email is unique and roles are constrained.
- `patients`: patient identity, address, and coordinates.
- `visits`: patient/caregiver assignments, scheduled and actual times, coordinates, status, and EVV result.

Visit foreign keys use `ON DELETE RESTRICT` to preserve history. The schema checks role/status values, coordinate ranges, and that scheduled end is later than scheduled start. The unique email constraint supplies the email index; visits also have focused indexes for caregiver, patient, start time, status, and EVV status.

## Configuration

Copy `.env.example` to `.env`. PostgreSQL is configured with `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_SSLMODE`. `DATABASE_URL` can be supplied instead and takes precedence.

`JWT_SECRET` is required by the API and signs access tokens. Replace the example value outside local development. Tokens expire after 24 hours; refresh tokens are not implemented.

## Running with Docker

```bash
cp .env.example .env
docker compose up --build -d postgres
docker compose run --rm backend careflow-migrate -direction up
docker compose up --build
```

The frontend is served at `http://localhost:4200`; the API and `GET /health` are at `http://localhost:8080`.

## Migrations

From a machine with Go installed:

```bash
cd backend
go run ./cmd/migrate -direction up
go run ./cmd/migrate -direction down
```

Or use Docker from the repository root:

```bash
docker compose run --rm backend careflow-migrate -direction up
docker compose run --rm backend careflow-migrate -direction down
```

`up` applies every pending migration. `down` rolls back the latest applied migration.

## Development Seed

After applying migrations, insert one fake coordinator, two fake caregivers, two fake patients, and three fake visits:

```bash
cd backend
go run ./cmd/seed
```

Or with Docker:

```bash
docker compose run --rm backend careflow-seed
```

The seed uses stable UUIDs and stores bcrypt hashes, never plaintext. It is transactional and safe to rerun.

Development-only credentials:

| Role | Email | Password |
| --- | --- | --- |
| Coordinator | `coordinator@careflow.local` | `careflow123` |
| Caregiver | `caregiver1@careflow.local` | `careflow123` |
| Caregiver | `caregiver2@careflow.local` | `careflow123` |

## Authentication

Log in:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"coordinator@careflow.local","password":"careflow123"}'
```

Use the returned token as a Bearer token:

```bash
curl http://localhost:8080/api/me \
  -H "Authorization: Bearer <token>"

curl http://localhost:8080/api/coordinator/ping \
  -H "Authorization: Bearer <token>"
```

Protected proof endpoints are `GET /api/me`, `GET /api/coordinator/ping`, and `GET /api/caregiver/ping`. The two ping endpoints enforce their roles strictly.

## Scheduling API

Coordinator-only endpoints:

- `POST /api/patients`
- `GET /api/patients`
- `GET /api/patients/{id}`
- `PUT /api/patients/{id}`
- `GET /api/caregivers`
- `POST /api/visits`
- `GET /api/visits`
- `GET /api/visits/{id}`
- `PATCH /api/visits/{id}`
- `POST /api/visits/{id}/cancel`

Caregiver-only endpoints:

- `GET /api/caregiver/visits`
- `GET /api/caregiver/visits/{id}`

Caregivers can see only visits assigned to the authenticated JWT user. They cannot access patient management or coordinator visit mutations.

Create a patient with a coordinator token:

```bash
curl -X POST http://localhost:8080/api/patients \
  -H "Authorization: Bearer <coordinator-token>" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Jane","last_name":"Smith","address":"100 Main Street, Baltimore, MD","latitude":39.2904,"longitude":-76.6122}'
```

Create a visit:

```bash
curl -X POST http://localhost:8080/api/visits \
  -H "Authorization: Bearer <coordinator-token>" \
  -H "Content-Type: application/json" \
  -d '{"patient_id":"<patient-uuid>","caregiver_id":"<caregiver-uuid>","scheduled_start":"2026-08-15T09:00:00-04:00","scheduled_end":"2026-08-15T12:00:00-04:00"}'
```

A caregiver cannot have overlapping scheduled or in-progress visits. The rule is `new_start < existing_end AND new_end > existing_start`, so visits that meet exactly at their boundaries are allowed. Cancelled visits do not cause conflicts.

## Running and Testing

```bash
cd frontend
npm install
npm start
```

```bash
cd backend
go run ./cmd/api
go test ./...
```

Repository integration tests require `TEST_DATABASE_URL` and create a unique temporary schema for each test group. With Docker:

```bash
docker compose run --rm backend-test
```

This command runs both repository and authentication HTTP integration tests against PostgreSQL. Authentication unit tests are included in the normal `go test ./...` run.

Run static checks with:

```bash
cd backend
go fmt ./...
go vet ./...
```

## Current Status

CareFlow is in core scheduling and caregiver-assignment development. Coordinator patient/visit management, caregiver lookup, overlap prevention, cancellation, and caregiver-owned visit reads are implemented. Frontend workflows and EVV check-in/check-out remain unimplemented.
