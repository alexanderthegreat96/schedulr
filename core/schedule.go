package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const calendarSearchDays = 366

var weekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "weds": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

type clockTime struct {
	hour, minute, second int
}

type parsedSchedule struct {
	times       []clockTime
	weekdays    map[time.Weekday]bool
	daysOfMonth map[int]bool
}

func (s parsedSchedule) dayMatches(day time.Time) bool {
	if len(s.weekdays) > 0 && !s.weekdays[day.Weekday()] {
		return false
	}

	if len(s.daysOfMonth) > 0 && !s.daysOfMonth[day.Day()] {
		return false
	}

	return true
}

func parseSchedule(e Execution) (parsedSchedule, error) {
	var s parsedSchedule

	for _, raw := range e.At {
		t, err := parseClockTime(raw)
		if err != nil {
			return s, err
		}
		s.times = append(s.times, t)
	}

	if len(s.times) == 0 {
		return s, fmt.Errorf("at is empty, so there is no time of day to run at")
	}

	sort.Slice(s.times, func(i, j int) bool {
		if s.times[i].hour != s.times[j].hour {
			return s.times[i].hour < s.times[j].hour
		}
		if s.times[i].minute != s.times[j].minute {
			return s.times[i].minute < s.times[j].minute
		}
		return s.times[i].second < s.times[j].second
	})

	if len(e.Days) > 0 {
		s.weekdays = make(map[time.Weekday]bool, len(e.Days))
		for _, raw := range e.Days {
			day, ok := weekdayNames[strings.ToLower(strings.TrimSpace(raw))]
			if !ok {
				return s, fmt.Errorf("%q is not a day of the week", raw)
			}
			s.weekdays[day] = true
		}
	}

	if len(e.DayOfMonth) > 0 {
		s.daysOfMonth = make(map[int]bool, len(e.DayOfMonth))
		for _, day := range e.DayOfMonth {
			if day < 1 || day > 31 {
				return s, fmt.Errorf("day_of_month %d is outside 1-31", day)
			}
			s.daysOfMonth[day] = true
		}
	}

	return s, nil
}

func parseClockTime(raw string) (clockTime, error) {
	trimmed := strings.TrimSpace(raw)

	for _, layout := range []string{"15:04", "15:04:05"} {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return clockTime{t.Hour(), t.Minute(), t.Second()}, nil
		}
	}

	return clockTime{}, fmt.Errorf("%q is not a time of day; use HH:MM or HH:MM:SS", raw)
}

func nextCalendarRun(after time.Time, e Execution) (time.Time, error) {
	schedule, err := parseSchedule(e)
	if err != nil {
		return time.Time{}, err
	}

	midnight := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location())

	for offset := 0; offset <= calendarSearchDays; offset++ {
		day := midnight.AddDate(0, 0, offset)

		if !schedule.dayMatches(day) {
			continue
		}

		for _, at := range schedule.times {
			candidate := time.Date(day.Year(), day.Month(), day.Day(), at.hour, at.minute, at.second, 0, day.Location())

			if candidate.After(after) {
				return candidate, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("no run time matches this schedule within %d days", calendarSearchDays)
}

func describeCalendarSchedule(e Execution, start time.Time) (string, time.Time) {
	nextRun, err := nextCalendarRun(start, e)
	if err != nil {
		return "invalid schedule: " + err.Error(), time.Time{}
	}

	description := "This task runs at " + strings.Join(e.At, ", ")

	if len(e.Days) > 0 {
		description += " on " + strings.Join(e.Days, ", ")
	}

	if len(e.DayOfMonth) > 0 {
		days := make([]string, 0, len(e.DayOfMonth))
		for _, d := range e.DayOfMonth {
			days = append(days, fmt.Sprintf("%d", d))
		}
		description += " on day " + strings.Join(days, ", ") + " of the month"
	}

	if len(e.Days) > 0 && len(e.DayOfMonth) > 0 {
		description += " (both must match)"
	}

	return description + ".", nextRun
}
