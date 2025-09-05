package sla

import (
	"testing"
	"time"
)

func testCalendar() *Calendar {
	loc, _ := time.LoadLocation("America/New_York")
	hrs := Hours{StartSec: 9 * 3600, EndSec: 17 * 3600}
	return &Calendar{
		Location: loc,
		Hours: map[time.Weekday]Hours{
			time.Monday:    hrs,
			time.Tuesday:   hrs,
			time.Wednesday: hrs,
			time.Thursday:  hrs,
			time.Friday:    hrs,
		},
		Holidays: map[time.Time]struct{}{},
	}
}

func TestBusinessDuration(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		holidays []time.Time
		want     time.Duration
	}{
		{
			name:  "on boundaries",
			start: time.Date(2024, 7, 1, 9, 0, 0, 0, loc),
			end:   time.Date(2024, 7, 1, 17, 0, 0, 0, loc),
			want:  8 * time.Hour,
		},
		{
			name:  "overnight",
			start: time.Date(2024, 7, 1, 16, 0, 0, 0, loc), // Mon 4pm
			end:   time.Date(2024, 7, 2, 10, 0, 0, 0, loc), // Tue 10am
			want:  2 * time.Hour,
		},
		{
			name:  "weekend span",
			start: time.Date(2024, 7, 5, 16, 0, 0, 0, loc), // Fri 4pm
			end:   time.Date(2024, 7, 8, 10, 0, 0, 0, loc), // Mon 10am
			want:  2 * time.Hour,
		},
		{
			name:  "consecutive holidays",
			start: time.Date(2024, 7, 2, 16, 0, 0, 0, loc), // Tue 4pm
			end:   time.Date(2024, 7, 5, 10, 0, 0, 0, loc), // Fri 10am
			holidays: []time.Time{
				time.Date(2024, 7, 3, 0, 0, 0, 0, loc),
				time.Date(2024, 7, 4, 0, 0, 0, 0, loc),
			},
			want: 2 * time.Hour,
		},
		{
			name:  "reversed inputs",
			start: time.Date(2024, 7, 2, 10, 0, 0, 0, loc), // Tue 10am
			end:   time.Date(2024, 7, 1, 16, 0, 0, 0, loc), // Mon 4pm
			want:  2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cal := testCalendar()
			for _, h := range tt.holidays {
				cal.Holidays[h.In(cal.Location)] = struct{}{}
			}
			got := cal.BusinessDuration(tt.start, tt.end)
			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}
