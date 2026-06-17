package validator

import "testing"

func TestCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		latitude  float64
		longitude float64
		wantError bool
	}{
		{name: "valid", latitude: -6.2, longitude: 106.8},
		{name: "north edge", latitude: 90, longitude: 0},
		{name: "latitude too high", latitude: 90.1, longitude: 0, wantError: true},
		{name: "longitude too low", latitude: 0, longitude: -180.1, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Coordinates(test.latitude, test.longitude)
			if (err != nil) != test.wantError {
				t.Fatalf("Coordinates(%v, %v) error = %v, wantError %v", test.latitude, test.longitude, err, test.wantError)
			}
		})
	}
}

func TestSafeFilename(t *testing.T) {
	got := SafeFilename("../../my holiday (1).jpg")
	if got != "my_holiday__1_.jpg" {
		t.Fatalf("SafeFilename() = %q", got)
	}
}
