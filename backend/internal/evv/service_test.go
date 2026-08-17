package evv

import (
	"math"
	"slices"
	"testing"
	"time"
)

func TestDistanceMeters(t *testing.T) {
	if distance := DistanceMeters(39.2904, -76.6122, 39.2904, -76.6122); distance > 0.01 {
		t.Fatalf("same coordinates should be approximately zero, got %.3f", distance)
	}
	nearby := DistanceMeters(39.2904, -76.6122, 39.2905, -76.6121)
	if nearby < 10 || nearby > 20 {
		t.Fatalf("expected nearby distance around 14 meters, got %.2f", nearby)
	}
	if far := DistanceMeters(39.2904, -76.6122, 39.3004, -76.6122); far <= 200 {
		t.Fatalf("expected far location outside geofence, got %.2f", far)
	}
}

func TestValidateTimeWindow(t *testing.T) {
	start := time.Date(2026, time.August, 15, 9, 0, 0, 0, time.UTC)
	tolerance := 15 * time.Minute
	tests := []struct {
		name   string
		actual time.Time
		want   string
	}{
		{name: "exact start", actual: start},
		{name: "exactly early boundary", actual: start.Add(-tolerance)},
		{name: "exactly late boundary", actual: start.Add(tolerance)},
		{name: "sixteen minutes early", actual: start.Add(-16 * time.Minute), want: ReasonEarlyCheckIn},
		{name: "sixteen minutes late", actual: start.Add(16 * time.Minute), want: ReasonLateCheckIn},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateTimeWindow(start, test.actual, tolerance); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestCheckInValidation(t *testing.T) {
	service := NewService(200, 15*time.Minute)
	start := time.Date(2026, time.August, 15, 9, 0, 0, 0, time.UTC)

	verified := service.ValidateCheckIn(39.2904, -76.6122, 39.2905, -76.6121, start, start)
	if verified.Status != StatusVerified || len(verified.Exceptions) != 0 {
		t.Fatalf("expected verified result, got %#v", verified)
	}

	multiple := service.ValidateCheckIn(39.2904, -76.6122, 39.3004, -76.6122, start, start.Add(16*time.Minute))
	if multiple.Status != StatusException ||
		!slices.Contains(multiple.Exceptions, ReasonLateCheckIn) ||
		!slices.Contains(multiple.Exceptions, ReasonOutsideGeofence) {
		t.Fatalf("expected timing and geofence exceptions, got %#v", multiple)
	}
}

func TestCheckOutPreservesAndAppendsExceptions(t *testing.T) {
	service := NewService(200, 15*time.Minute)
	result := service.ValidateCheckOut(
		39.2904, -76.6122, 39.3004, -76.6122,
		[]string{ReasonLateCheckIn},
	)
	if result.Status != StatusException ||
		!slices.Contains(result.Exceptions, ReasonLateCheckIn) ||
		!slices.Contains(result.Exceptions, ReasonCheckoutOutsideGeofence) {
		t.Fatalf("expected preserved and checkout exceptions, got %#v", result)
	}
	if math.IsNaN(result.DistanceMeters) || result.DistanceMeters <= 200 {
		t.Fatalf("unexpected checkout distance %.2f", result.DistanceMeters)
	}
}
