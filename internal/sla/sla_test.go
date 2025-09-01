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

func TestBusinessDurationBasic(t *testing.T) {
	cal := testCalendar()
	loc := cal.Location
	start := time.Date(2024, 7, 1, 16, 0, 0, 0, loc) // Mon 4pm
	end := time.Date(2024, 7, 2, 10, 0, 0, 0, loc)   // Tue 10am
	d := cal.BusinessDuration(start, end)
	if d != 2*time.Hour {
		t.Fatalf("expected 2h got %v", d)
	}
}

func TestBusinessDurationHoliday(t *testing.T) {
	cal := testCalendar()
	loc := cal.Location
	holiday := time.Date(2024, 7, 4, 0, 0, 0, 0, loc)
	cal.Holidays[holiday] = struct{}{}
	start := time.Date(2024, 7, 3, 16, 0, 0, 0, loc)
	end := time.Date(2024, 7, 5, 10, 0, 0, 0, loc)
	d := cal.BusinessDuration(start, end)
	if d != 2*time.Hour {
		t.Fatalf("expected 2h got %v", d)
	}
}

