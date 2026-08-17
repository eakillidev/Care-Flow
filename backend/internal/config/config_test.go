package config

import "testing"

func TestLoadEVVDefaults(t *testing.T) {
	t.Setenv("EVV_GEOFENCE_METERS", "")
	t.Setenv("EVV_TIME_TOLERANCE_MINUTES", "")
	config, err := Load()
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if config.EVVGeofenceMeters != 200 {
		t.Fatalf("expected 200 meter default, got %v", config.EVVGeofenceMeters)
	}
}

func TestLoadRejectsInvalidEVVConfiguration(t *testing.T) {
	for _, test := range []struct{ key, value string }{
		{key: "EVV_GEOFENCE_METERS", value: "-1"},
		{key: "EVV_TIME_TOLERANCE_MINUTES", value: "0"},
		{key: "EVV_GEOFENCE_METERS", value: "invalid"},
	} {
		t.Run(test.key+test.value, func(t *testing.T) {
			t.Setenv("EVV_GEOFENCE_METERS", "200")
			t.Setenv("EVV_TIME_TOLERANCE_MINUTES", "15")
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil {
				t.Fatal("expected invalid EVV configuration to fail")
			}
		})
	}
}
