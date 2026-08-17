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
	Latitude  float64   `json:"-"`
	Longitude float64   `json:"-"`
}

type EVVPoint struct {
	Timestamp                 *time.Time `json:"timestamp"`
	Latitude                  *float64   `json:"latitude"`
	Longitude                 *float64   `json:"longitude"`
	DistanceFromPatientMeters *float64   `json:"distance_from_patient_meters"`
}

type EVVDetail struct {
	Status           EVVStatus `json:"status"`
	ExceptionReasons []string  `json:"exception_reasons"`
	CheckIn          EVVPoint  `json:"check_in"`
	CheckOut         EVVPoint  `json:"check_out"`
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
	EVV               EVVDetail        `json:"evv"`
}

type Filter struct {
	Status      *Status
	EVVStatus   *EVVStatus
	CaregiverID *uuid.UUID
	PatientID   *uuid.UUID
	From        *time.Time
	To          *time.Time
}

type Summary struct {
	TotalVisits   int64 `json:"total_visits"`
	Scheduled     int64 `json:"scheduled"`
	InProgress    int64 `json:"in_progress"`
	Completed     int64 `json:"completed"`
	Cancelled     int64 `json:"cancelled"`
	EVVVerified   int64 `json:"evv_verified"`
	EVVExceptions int64 `json:"evv_exceptions"`
}
