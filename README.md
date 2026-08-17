# CareFlow

CareFlow is a full-stack homecare visit management platform for caregiver scheduling, Electronic Visit Verification (EVV), and visit exception monitoring.

## Overview

Coordinators manage patients, schedule visits, assign caregivers, and review visit and EVV exception status. Caregivers see only their assigned visits and use browser geolocation to check in and out. The Go API validates ownership, visit state, scheduling conflicts, time tolerance, and proximity to the patient's location.

## Tech Stack

- **Frontend:** Angular 21, TypeScript
- **Backend:** Go 1.24, chi
- **Database:** PostgreSQL 17, pgx/pgxpool
- **Security:** bcrypt, JWT, role-based authorization
- **Development and quality:** Docker, Docker Compose, GitHub Actions, Angular tests, Go unit tests, PostgreSQL integration tests

## Core Features

- Coordinator and caregiver authentication
- Patient management and caregiver assignment
- Visit scheduling and overlap detection
- Role-based and ownership-based access control
- GPS-based EVV check-in and check-out
- Configurable geofence and time-window validation
- Stable EVV exception handling
- Coordinator monitoring and EVV review dashboard
- Caregiver assignment and visit-action dashboard

## Architecture

```text
Angular + TypeScript
        |
        v
      Go API
        |
        v
   PostgreSQL
```

HTTP handlers, business services, EVV rules, and repositories have separate responsibilities. Database access remains isolated behind explicit pgx repositories and parameterized SQL. See [ARCHITECTURE.md](ARCHITECTURE.md) for the key decisions.

## EVV Workflow

```text
Coordinator schedules visit
        |
        v
Caregiver receives assignment
        |
        v
Caregiver checks in
        |
        v
Go API validates time + location
        |
        v
Verified or EVV Exception
        |
        v
Caregiver checks out
        |
        v
Coordinator reviews completed visit
```

The API uses server-generated timestamps. Check-in is valid within the configured time tolerance and geofence; violations are recorded as EVV exceptions rather than silently discarded. Conditional PostgreSQL updates protect check-in and check-out state transitions from duplicate concurrent requests.

## Project Structure

```text
Care-Flow/
|-- frontend/                 Angular application
|-- backend/
|   |-- cmd/                  API, migration, and seed commands
|   |-- internal/             Auth, domain services, EVV, and repositories
|   `-- migrations/           Versioned PostgreSQL migrations
|-- docs/images/              Portfolio screenshot location
|-- .github/workflows/        Continuous integration
|-- ARCHITECTURE.md
`-- docker-compose.yml
```

## Local Setup

```bash
cp .env.example .env
docker compose up --build -d
docker compose run --rm backend careflow-migrate -direction up
docker compose run --rm backend careflow-seed
```

- Frontend: `http://localhost:4200/login`
- API: `http://localhost:8080`
- Health check: `http://localhost:8080/health`

Development-only credentials:

| Role | Email | Password |
| --- | --- | --- |
| Coordinator | `coordinator@careflow.local` | `careflow123` |
| Caregiver | `caregiver1@careflow.local` | `careflow123` |
| Caregiver | `caregiver2@careflow.local` | `careflow123` |

Browser location permission is required for caregiver EVV actions. Local defaults in `.env.example` are for development only; replace `JWT_SECRET` and database credentials outside local use.

## Testing

```bash
# Angular tests and production build
cd frontend
npm ci
npm test -- --watch=false
npm run build

# Go unit tests and static analysis
cd ../backend
gofmt -l .
go vet ./...
go test ./...

# PostgreSQL integration tests and Compose validation, from repository root
cd ..
docker compose up -d postgres
docker compose run --rm backend-test
docker compose config --quiet
```

Integration tests create isolated PostgreSQL schemas and clean them up independently.

## API Highlights

- `POST /api/auth/login`
- `GET /api/patients`
- `POST /api/visits`
- `GET /api/caregiver/visits`
- `POST /api/caregiver/visits/{id}/check-in`
- `POST /api/caregiver/visits/{id}/check-out`

Coordinator visit endpoints also support combined status, EVV, caregiver, patient, and date filters plus an aggregate EVV summary.

## Screenshots

### Coordinator Dashboard
<!-- Add coordinator dashboard screenshot to docs/images/ when available. -->

### Caregiver Dashboard
<!-- Add caregiver dashboard screenshot to docs/images/ when available. -->

### EVV Visit Detail
<!-- Add EVV visit detail screenshot to docs/images/ when available. -->

## Project Status

CareFlow is a completed portfolio-scale application demonstrating a tested end-to-end homecare scheduling and EVV workflow. It is not presented as production software.
