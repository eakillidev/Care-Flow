CREATE TABLE users (
    id UUID PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(320) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_role_check CHECK (role IN ('caregiver', 'coordinator'))
);

CREATE TABLE patients (
    id UUID PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    address VARCHAR(500) NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT patients_latitude_check CHECK (latitude BETWEEN -90 AND 90),
    CONSTRAINT patients_longitude_check CHECK (longitude BETWEEN -180 AND 180)
);

CREATE TABLE visits (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL,
    caregiver_id UUID NOT NULL,
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    actual_check_in TIMESTAMPTZ,
    actual_check_out TIMESTAMPTZ,
    check_in_latitude DOUBLE PRECISION,
    check_in_longitude DOUBLE PRECISION,
    check_out_latitude DOUBLE PRECISION,
    check_out_longitude DOUBLE PRECISION,
    status VARCHAR(20) NOT NULL,
    evv_status VARCHAR(20) NOT NULL,
    evv_exception TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT visits_patient_fk FOREIGN KEY (patient_id)
        REFERENCES patients (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT visits_caregiver_fk FOREIGN KEY (caregiver_id)
        REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT visits_schedule_check CHECK (scheduled_end > scheduled_start),
    CONSTRAINT visits_status_check CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    CONSTRAINT visits_evv_status_check CHECK (evv_status IN ('pending', 'verified', 'exception')),
    CONSTRAINT visits_check_in_latitude_check CHECK (check_in_latitude IS NULL OR check_in_latitude BETWEEN -90 AND 90),
    CONSTRAINT visits_check_in_longitude_check CHECK (check_in_longitude IS NULL OR check_in_longitude BETWEEN -180 AND 180),
    CONSTRAINT visits_check_out_latitude_check CHECK (check_out_latitude IS NULL OR check_out_latitude BETWEEN -90 AND 90),
    CONSTRAINT visits_check_out_longitude_check CHECK (check_out_longitude IS NULL OR check_out_longitude BETWEEN -180 AND 180)
);

CREATE INDEX visits_caregiver_id_idx ON visits (caregiver_id);
CREATE INDEX visits_patient_id_idx ON visits (patient_id);
CREATE INDEX visits_scheduled_start_idx ON visits (scheduled_start);
CREATE INDEX visits_status_idx ON visits (status);
CREATE INDEX visits_evv_status_idx ON visits (evv_status);
