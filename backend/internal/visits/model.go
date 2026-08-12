package visits

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusScheduled  Status = "scheduled"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

type EVVStatus string

const (
	EVVStatusPending   EVVStatus = "pending"
	EVVStatusVerified  EVVStatus = "verified"
	EVVStatusException EVVStatus = "exception"
)

type Visit struct {
	ID                uuid.UUID  `json:"id"`
	PatientID         uuid.UUID  `json:"patient_id"`
	CaregiverID       uuid.UUID  `json:"caregiver_id"`
	ScheduledStart    time.Time  `json:"scheduled_start"`
	ScheduledEnd      time.Time  `json:"scheduled_end"`
	ActualCheckIn     *time.Time `json:"actual_check_in,omitempty"`
	ActualCheckOut    *time.Time `json:"actual_check_out,omitempty"`
	CheckInLatitude   *float64   `json:"check_in_latitude,omitempty"`
	CheckInLongitude  *float64   `json:"check_in_longitude,omitempty"`
	CheckOutLatitude  *float64   `json:"check_out_latitude,omitempty"`
	CheckOutLongitude *float64   `json:"check_out_longitude,omitempty"`
	Status            Status     `json:"status"`
	EVVStatus         EVVStatus  `json:"evv_status"`
	EVVException      *string    `json:"evv_exception,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PatientSummary struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Address   string    `json:"address"`
}

type CaregiverSummary struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

type Detail struct {
	ID                uuid.UUID        `json:"id"`
	Patient           PatientSummary   `json:"patient"`
	Caregiver         CaregiverSummary `json:"caregiver"`
	ScheduledStart    time.Time        `json:"scheduled_start"`
	ScheduledEnd      time.Time        `json:"scheduled_end"`
	ActualCheckIn     *time.Time       `json:"actual_check_in"`
	ActualCheckOut    *time.Time       `json:"actual_check_out"`
	CheckInLatitude   *float64         `json:"check_in_latitude"`
	CheckInLongitude  *float64         `json:"check_in_longitude"`
	CheckOutLatitude  *float64         `json:"check_out_latitude"`
	CheckOutLongitude *float64         `json:"check_out_longitude"`
	Status            Status           `json:"status"`
	EVVStatus         EVVStatus        `json:"evv_status"`
	EVVException      *string          `json:"evv_exception"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}
