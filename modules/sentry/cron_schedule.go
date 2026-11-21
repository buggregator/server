package sentry

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the alpine image has no zoneinfo: monitor time zones need it

	"github.com/robfig/cron/v3"
)

// cronSchedule is a monitor's schedule as Sentry's monitor_config sends it:
// {"type": "crontab", "value": "0 3 * * *"} or
// {"type": "interval", "value": 10, "unit": "minute"}. A zero value means no
// schedule: check-ins are recorded, slots are never missed.
type cronSchedule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

const (
	scheduleCrontab  = "crontab"
	scheduleInterval = "interval"
)

var errBadSchedule = errors.New("invalid schedule")

// nextFunc returns the first slot strictly after t, zero when there is none.
type nextFunc func(t time.Time) time.Time

// compile checks the schedule and the time zone and returns its next-slot
// function. A zero schedule compiles to nil.
func (s cronSchedule) compile(tz string) (nextFunc, error) {
	loc := time.UTC
	if tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("%w: unknown time zone %q", errBadSchedule, tz)
		}
		loc = l
	}
	switch s.Type {
	case "":
		return nil, nil
	case scheduleCrontab:
		spec := strings.TrimSpace(s.Value)
		if strings.HasPrefix(spec, "@every") || strings.Contains(spec, "TZ=") {
			return nil, fmt.Errorf("%w: crontab %q", errBadSchedule, s.Value)
		}
		sched, err := cron.ParseStandard(spec)
		if err != nil {
			return nil, fmt.Errorf("%w: crontab %q: %v", errBadSchedule, s.Value, err)
		}
		return func(t time.Time) time.Time {
			n := sched.Next(t.In(loc))
			// When clocks go back, a wall time repeats: cron runs a job once,
			// so the second occurrence is no slot.
			for i := 0; i < 4 && !n.IsZero() && sameWallClock(n, n.Add(-time.Hour), loc); i++ {
				n = sched.Next(n)
			}
			return n.UTC()
		}, nil
	case scheduleInterval:
		n, err := strconv.Atoi(strings.TrimSpace(s.Value))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%w: interval value %q", errBadSchedule, s.Value)
		}
		var add func(t time.Time) time.Time
		switch strings.TrimSuffix(s.Unit, "s") {
		case "year":
			add = func(t time.Time) time.Time { return t.AddDate(n, 0, 0) }
		case "month":
			add = func(t time.Time) time.Time { return t.AddDate(0, n, 0) }
		case "week":
			add = func(t time.Time) time.Time { return t.AddDate(0, 0, 7*n) }
		case "day":
			add = func(t time.Time) time.Time { return t.AddDate(0, 0, n) }
		case "hour":
			add = func(t time.Time) time.Time { return t.Add(time.Duration(n) * time.Hour) }
		case "minute":
			add = func(t time.Time) time.Time { return t.Add(time.Duration(n) * time.Minute) }
		default:
			return nil, fmt.Errorf("%w: interval unit %q", errBadSchedule, s.Unit)
		}
		return func(t time.Time) time.Time { return add(t.In(loc)).UTC() }, nil
	}
	return nil, fmt.Errorf("%w: type %q", errBadSchedule, s.Type)
}

func sameWallClock(a, b time.Time, loc *time.Location) bool {
	a, b = a.In(loc), b.In(loc)
	return a.Hour() == b.Hour() && a.Minute() == b.Minute() && a.YearDay() == b.YearDay()
}

// cronTimeFormat is how cron tables store times: UTC, fixed width, so text
// order is time order.
const cronTimeFormat = "2006-01-02T15:04:05.000Z"

func cronTime(t time.Time) string { return t.UTC().Format(cronTimeFormat) }

func parseCronTime(s string) time.Time {
	t, _ := time.Parse(cronTimeFormat, s)
	return t
}
