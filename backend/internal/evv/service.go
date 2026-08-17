package evv

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	ReasonEarlyCheckIn            = "early_check_in"
	ReasonLateCheckIn             = "late_check_in"
	ReasonOutsideGeofence         = "outside_geofence"
	ReasonCheckoutOutsideGeofence = "checkout_outside_geofence"
)

type Status string

const (
	StatusVerified  Status = "verified"
	StatusException Status = "exception"
)

type Result struct {
	Status         Status   `json:"status"`
	DistanceMeters float64  `json:"distance_meters"`
	Exceptions     []string `json:"exceptions"`
}

type Service struct {
	geofenceMeters float64
	timeTolerance  time.Duration
}

func NewService(geofenceMeters float64, timeTolerance time.Duration) *Service {
	return &Service{geofenceMeters: geofenceMeters, timeTolerance: timeTolerance}
}

func (service *Service) ValidateCheckIn(
	patientLatitude, patientLongitude, caregiverLatitude, caregiverLongitude float64,
	scheduledStart, checkedInAt time.Time,
) Result {
	distance := DistanceMeters(patientLatitude, patientLongitude, caregiverLatitude, caregiverLongitude)
	exceptions := make([]string, 0, 2)
	if timingReason := ValidateTimeWindow(scheduledStart, checkedInAt, service.timeTolerance); timingReason != "" {
		exceptions = append(exceptions, timingReason)
	}
	if distance > service.geofenceMeters {
		exceptions = append(exceptions, ReasonOutsideGeofence)
	}
	return result(distance, exceptions)
}

func (service *Service) ValidateCheckOut(
	patientLatitude, patientLongitude, caregiverLatitude, caregiverLongitude float64,
	previousExceptions []string,
) Result {
	distance := DistanceMeters(patientLatitude, patientLongitude, caregiverLatitude, caregiverLongitude)
	exceptions := append([]string(nil), previousExceptions...)
	if distance > service.geofenceMeters {
		exceptions = append(exceptions, ReasonCheckoutOutsideGeofence)
	}
	return result(distance, unique(exceptions))
}

func DistanceMeters(latitudeOne, longitudeOne, latitudeTwo, longitudeTwo float64) float64 {
	const earthRadiusMeters = 6371000.0
	latOne := latitudeOne * math.Pi / 180
	latTwo := latitudeTwo * math.Pi / 180
	deltaLatitude := (latitudeTwo - latitudeOne) * math.Pi / 180
	deltaLongitude := (longitudeTwo - longitudeOne) * math.Pi / 180
	a := math.Sin(deltaLatitude/2)*math.Sin(deltaLatitude/2) +
		math.Cos(latOne)*math.Cos(latTwo)*math.Sin(deltaLongitude/2)*math.Sin(deltaLongitude/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func ValidateTimeWindow(scheduledStart, checkedInAt time.Time, tolerance time.Duration) string {
	if checkedInAt.Before(scheduledStart.Add(-tolerance)) {
		return ReasonEarlyCheckIn
	}
	if checkedInAt.After(scheduledStart.Add(tolerance)) {
		return ReasonLateCheckIn
	}
	return ""
}

func ParseExceptions(value *string) []string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return []string{}
	}
	parts := strings.Split(*value, ",")
	return unique(parts)
}

func JoinExceptions(exceptions []string) *string {
	if len(exceptions) == 0 {
		return nil
	}
	value := strings.Join(unique(exceptions), ",")
	return &value
}

func result(distance float64, exceptions []string) Result {
	status := StatusVerified
	if len(exceptions) > 0 {
		status = StatusException
	}
	return Result{Status: status, DistanceMeters: distance, Exceptions: exceptions}
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
