/*
Copyright (C) 2026 Berlian Bima Seto

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package bannedbook

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var errdate = errors.New("unrecognised date, try tomorrow, mon, +3d, 2026-11-10 or 2026-11-10 14:00")

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

func midnight(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
}

func dayof(word string, base time.Time) (time.Time, bool) {
	switch word {
	case "today":
		return base, true
	case "tomorrow", "tmr":
		return base.AddDate(0, 0, 1), true
	}

	if day, ok := weekdays[word]; ok {
		delta := (int(day) - int(base.Weekday()) + 7) % 7
		if delta == 0 {
			delta = 7
		}
		return base.AddDate(0, 0, delta), true
	}

	if strings.HasPrefix(word, "+") && len(word) > 2 {
		count, err := strconv.Atoi(word[1 : len(word)-1])
		if err != nil || count < 0 || count > 3650 {
			return time.Time{}, false
		}

		switch word[len(word)-1] {
		case 'd':
			return base.AddDate(0, 0, count), true
		case 'w':
			return base.AddDate(0, 0, 7*count), true
		}

		return time.Time{}, false
	}

	if parsed, err := time.ParseInLocation("2006-01-02", word, base.Location()); err == nil {
		return parsed, true
	}

	return time.Time{}, false
}

func timeof(word string) (time.Duration, bool) {
	for _, layout := range []string{"15:04", "3pm", "3:04pm"} {
		if parsed, err := time.Parse(layout, word); err == nil {
			return time.Duration(parsed.Hour())*time.Hour + time.Duration(parsed.Minute())*time.Minute, true
		}
	}

	return 0, false
}

func moment(input string, now time.Time) (time.Time, bool, error) {
	words := strings.Fields(strings.ToLower(input))
	if len(words) == 0 || len(words) > 2 {
		return time.Time{}, false, errdate
	}

	base := midnight(now)

	var day time.Time
	var clock time.Duration
	var hasday, hasclock bool

	for _, word := range words {
		if found, ok := dayof(word, base); ok && !hasday {
			day, hasday = found, true
			continue
		}

		if found, ok := timeof(word); ok && !hasclock {
			clock, hasclock = found, true
			continue
		}

		return time.Time{}, false, errdate
	}

	if !hasday {
		day = base
	}

	if day.Before(base) {
		return time.Time{}, false, errors.New("that date is in the past")
	}

	return day.Add(clock), hasclock, nil
}
