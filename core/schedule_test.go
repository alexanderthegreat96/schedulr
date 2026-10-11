package core

import (
	"strings"
	"testing"
	"time"
)

var refNow = time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC)

func TestNextCalendarRunDailyTime(t *testing.T) {
	t.Parallel()

	next, err := nextCalendarRun(refNow, Execution{At: []string{"03:00"}})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want := time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextCalendarRunLaterToday(t *testing.T) {
	t.Parallel()

	next, err := nextCalendarRun(refNow, Execution{At: []string{"23:15"}})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want := time.Date(2026, 3, 4, 23, 15, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextCalendarRunPicksTheEarliestFutureTime(t *testing.T) {
	t.Parallel()

	e := Execution{At: []string{"17:00", "09:00", "13:00"}}

	next, err := nextCalendarRun(refNow, e)
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want := time.Date(2026, 3, 4, 13, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v (13:00 is the next one after 10:30)", next, want)
	}

	after := time.Date(2026, 3, 4, 17, 0, 1, 0, time.UTC)
	next, err = nextCalendarRun(after, e)
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want = time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextCalendarRunWeekdayFilter(t *testing.T) {
	t.Parallel()

	next, err := nextCalendarRun(refNow, Execution{At: []string{"09:00"}, Days: []string{"mon"}})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
	if next.Weekday() != time.Monday {
		t.Errorf("next lands on %v, want Monday", next.Weekday())
	}
}

func TestNextCalendarRunWeekdayNamesAreFlexible(t *testing.T) {
	t.Parallel()

	for _, spelling := range [][]string{
		{"mon"}, {"Mon"}, {"MONDAY"}, {"monday"}, {" mon "},
	} {
		next, err := nextCalendarRun(refNow, Execution{At: []string{"09:00"}, Days: spelling})
		if err != nil {
			t.Fatalf("%v: %v", spelling, err)
		}
		if next.Weekday() != time.Monday {
			t.Errorf("%v gave %v", spelling, next.Weekday())
		}
	}
}

func TestNextCalendarRunDayOfMonthFilter(t *testing.T) {
	t.Parallel()

	next, err := nextCalendarRun(refNow, Execution{At: []string{"06:00"}, DayOfMonth: []int{15}})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	want := time.Date(2026, 3, 15, 6, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextCalendarRunCombinesDayFiltersWithAnd(t *testing.T) {
	t.Parallel()

	next, err := nextCalendarRun(refNow, Execution{
		At:         []string{"00:30"},
		Days:       []string{"thu"},
		DayOfMonth: []int{1},
	})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	if next.Day() != 1 || next.Weekday() != time.Thursday {
		t.Errorf("next = %v, want a day that is both the 1st and a Thursday", next)
	}
	if want := time.Date(2026, 10, 1, 0, 30, 0, 0, time.UTC); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestNextCalendarRunRejectsAnImpossibleSchedule(t *testing.T) {
	t.Parallel()

	_, err := nextCalendarRun(refNow, Execution{
		At:         []string{"03:00"},
		Days:       []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"},
		DayOfMonth: []int{32},
	})

	if err == nil {
		t.Fatal("day_of_month 32 should be rejected")
	}
	if !strings.Contains(err.Error(), "1-31") {
		t.Errorf("error should explain the valid range, got: %v", err)
	}
}

func TestParseClockTimeAcceptsSecondsAndRejectsJunk(t *testing.T) {
	t.Parallel()

	good := map[string]clockTime{
		"00:00":    {0, 0, 0},
		"3:05":     {3, 5, 0},
		"03:05":    {3, 5, 0},
		"23:59":    {23, 59, 0},
		"06:30:45": {6, 30, 45},
		" 09:15 ":  {9, 15, 0},
	}

	for raw, want := range good {
		got, err := parseClockTime(raw)
		if err != nil {
			t.Errorf("parseClockTime(%q) failed: %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("parseClockTime(%q) = %+v, want %+v", raw, got, want)
		}
	}

	for _, raw := range []string{"", "25:00", "12:60", "noon", "3pm", "03", "1:2:3:4"} {
		if _, err := parseClockTime(raw); err == nil {
			t.Errorf("parseClockTime(%q) should have failed", raw)
		}
	}
}

func TestParseScheduleRejectsBadInput(t *testing.T) {
	t.Parallel()

	tests := map[string]Execution{
		"no times":        {At: []string{}},
		"bad time":        {At: []string{"half past three"}},
		"bad day":         {At: []string{"03:00"}, Days: []string{"funday"}},
		"day of month 0":  {At: []string{"03:00"}, DayOfMonth: []int{0}},
		"day of month 32": {At: []string{"03:00"}, DayOfMonth: []int{32}},
	}

	for name, e := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSchedule(e); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestNextCalendarRunIsStrictlyAfter(t *testing.T) {
	t.Parallel()

	exactly := time.Date(2026, 3, 4, 3, 0, 0, 0, time.UTC)

	next, err := nextCalendarRun(exactly, Execution{At: []string{"03:00"}})
	if err != nil {
		t.Fatalf("nextCalendarRun failed: %v", err)
	}

	if !next.After(exactly) {
		t.Errorf("next = %v, should be strictly after %v", next, exactly)
	}
	if want := time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestHasCalendarSchedule(t *testing.T) {
	t.Parallel()

	if (Execution{}).HasCalendarSchedule() {
		t.Error("an execution with no at times is not on a calendar schedule")
	}
	if !(Execution{At: []string{"03:00"}}).HasCalendarSchedule() {
		t.Error("an at time means a calendar schedule")
	}
	if (Execution{Days: []string{"mon"}}).HasCalendarSchedule() {
		t.Error("days without at is not a calendar schedule")
	}
}
