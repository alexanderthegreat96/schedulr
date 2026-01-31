package core

import (
	"testing"
	"time"
)

func TestDefaultValueIfNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    any
		typeName string
		expected any
	}{
		{"nil string", nil, "string", "no-value"},
		{"nil int", nil, "int", 0},
		{"nil float64", nil, "float64", 0.0},
		{"nil bool", nil, "bool", false},
		{"nil map", nil, "map", map[string]any{}},
		{"nil slice", nil, "slice", []any{}},
		{"nil any", nil, "any", "(unset)"},
		{"empty string", "", "string", "no-value"},
		{"zero int", 0, "int", 0},
		{"zero float64", 0.0, "float64", 0.0},
		{"non-empty string", "hello", "string", "hello"},
		{"non-zero int", 42, "int", 42},
		{"non-empty map", map[string]any{"key": "value"}, "map", map[string]any{"key": "value"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DefaultValueIfNull(tc.input, tc.typeName)
			if tc.typeName == "map" {
				if _, ok := result.(map[string]any); !ok {
					t.Errorf("expected map[string]any, got %T", result)
				}
			} else if tc.typeName == "slice" {
				if _, ok := result.([]any); !ok {
					t.Errorf("expected []any, got %T", result)
				}
			} else if result != tc.expected {
				t.Errorf("DefaultValueIfNull(%v, %q) = %v, want %v", tc.input, tc.typeName, result, tc.expected)
			}
		})
	}
}

func TestBuildIntervalParts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		interval Interval
		expected []string
	}{
		{
			"single value",
			Interval{Years: 1},
			[]string{"1 year"},
		},
		{
			"multiple values",
			Interval{Years: 2, Months: 3, Days: 5},
			[]string{"2 years", "3 months", "5 days"},
		},
		{
			"empty interval",
			Interval{},
			[]string{},
		},
		{
			"all values",
			Interval{Years: 1, Months: 2, Weeks: 1, Days: 1, Hours: 3, Minutes: 30, Seconds: 45},
			[]string{"1 year", "2 months", "1 week", "1 day", "3 hours", "30 minutes", "45 seconds"},
		},
		{
			"plural singular mix",
			Interval{Years: 1, Months: 2},
			[]string{"1 year", "2 months"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildIntervalParts(tc.interval)
			if len(result) != len(tc.expected) {
				t.Errorf("got %d parts, want %d", len(result), len(tc.expected))
				return
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("part %d: got %q, want %q", i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestFormatParts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{"single part", []string{"daily"}, "daily"},
		{"two parts", []string{"1 day", "2 hours"}, "1 day and 2 hours"},
		{"three parts", []string{"1 day", "2 hours", "30 minutes"}, "1 day, 2 hours, and 30 minutes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatParts(tc.parts)
			if result != tc.expected {
				t.Errorf("formatParts(%v) = %q, want %q", tc.parts, result, tc.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    int
		unit     string
		expected string
	}{
		{"singular", 1, "day", "1 day"},
		{"plural", 5, "day", "5 days"},
		{"plural month", 2, "month", "2 months"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := pluralize(tc.value, tc.unit)
			if result != tc.expected {
				t.Errorf("pluralize(%d, %q) = %q, want %q", tc.value, tc.unit, result, tc.expected)
			}
		})
	}
}

func TestDescribeInterval(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		interval      Interval
		intervalType  string
		shouldContain string
	}{
		{"delay description", Interval{Hours: 2}, "delay", "delay of 2 hours"},
		{"interval description", Interval{Days: 1}, "interval", "every 1 day"},
		{"empty delay", Interval{}, "delay", "no delay"},
		{"empty interval", Interval{}, "interval", "immediately"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desc, _ := DescribeInterval(tc.interval, start, tc.intervalType)
			if desc != tc.shouldContain {
				t.Errorf("got description %q, want %q", desc, tc.shouldContain)
			}
		})
	}
}

func TestGetFirstAndNextRun(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	delay := Interval{Hours: 1}
	interval := Interval{Days: 1}

	first, next := GetFirstAndNextRun(start, time.Time{}, delay, interval)

	expected := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	if !first.Equal(expected) {
		t.Errorf("first run = %v, want %v", first, expected)
	}

	expectedNext := time.Date(2024, 1, 2, 11, 0, 0, 0, time.UTC)
	if !next.Equal(expectedNext) {
		t.Errorf("next run = %v, want %v", next, expectedNext)
	}
}

func TestIsZeroInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		interval Interval
		expected bool
	}{
		{"all zero", Interval{}, true},
		{"one value", Interval{Days: 1}, false},
		{"multiple values", Interval{Hours: 1, Minutes: 30}, false},
		{"seconds only", Interval{Seconds: 30}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isZeroInterval(tc.interval)
			if result != tc.expected {
				t.Errorf("isZeroInterval(%v) = %v, want %v", tc.interval, result, tc.expected)
			}
		})
	}
}

func TestDescribeSchedule(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	exec := Execution{
		Delay:    Interval{Hours: 1},
		Interval: Interval{Days: 1},
	}

	desc, _ := DescribeSchedule(exec, start)

	if desc == "" {
		t.Error("expected non-empty description")
	}

	if !contains(desc, "runs") {
		t.Errorf("description should mention 'runs', got: %s", desc)
	}
}

func TestCalculateNextRun(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		interval Interval
		expected time.Time
	}{
		{
			"add hours",
			Interval{Hours: 2},
			time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			"add days",
			Interval{Days: 1},
			time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			"add months",
			Interval{Months: 1},
			time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			"zero interval",
			Interval{},
			start,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateNextRun(start, tc.interval)
			if !result.Equal(tc.expected) {
				t.Errorf("calculateNextRun() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
