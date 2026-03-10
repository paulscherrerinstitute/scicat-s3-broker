package scicat

import (
	"testing"
	"time"
)

func TestMinTime(t *testing.T) {
	now := time.Now()
	later := now.Add(2 * time.Hour)
	zero := time.Time{}

	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected time.Time
	}{
		{"t1 is zero", zero, now, now},
		{"t2 is zero", now, zero, now},
		{"both are zero", zero, zero, zero},
		{"t1 is earlier", now, later, now},
		{"t2 is earlier", later, now, now},
		{"times are equal", now, now, now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minTime(tt.t1, tt.t2)

			if !got.Equal(tt.expected) {
				t.Errorf("minTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}
