package sla

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type Hours struct {
	StartSec int
	EndSec   int
}

type Calendar struct {
	Location *time.Location
	Hours    map[time.Weekday]Hours
	Holidays map[time.Time]struct{}
}

func LoadCalendar(ctx context.Context, db DB, id string) (*Calendar, error) {
	var tz string
	if err := db.QueryRow(ctx, "select tz from calendars where id=$1", id).Scan(&tz); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, err
	}
	cal := &Calendar{
		Location: loc,
		Hours:    make(map[time.Weekday]Hours),
		Holidays: make(map[time.Time]struct{}),
	}
	rows, err := db.Query(ctx, "select dow, start_sec, end_sec from business_hours where calendar_id=$1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var dow, start, end int
		if err := rows.Scan(&dow, &start, &end); err == nil {
			cal.Hours[time.Weekday(dow)] = Hours{StartSec: start, EndSec: end}
		}
	}
	hrows, err := db.Query(ctx, "select date from holidays where calendar_id=$1", id)
	if err != nil {
		return nil, err
	}
	defer hrows.Close()
	for hrows.Next() {
		var d time.Time
		if err := hrows.Scan(&d); err == nil {
			day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
			cal.Holidays[day] = struct{}{}
		}
	}
	return cal, nil
}

func (c *Calendar) BusinessDuration(start, end time.Time) time.Duration {
	if end.Before(start) {
		start, end = end, start
	}
	start = start.In(c.Location)
	end = end.In(c.Location)
	total := time.Duration(0)
	cur := start
	for cur.Before(end) {
		dayStart := time.Date(cur.Year(), cur.Month(), cur.Day(), 0, 0, 0, 0, c.Location)
		dayEnd := dayStart.Add(24 * time.Hour)
		if _, ok := c.Holidays[dayStart]; ok {
			cur = dayEnd
			continue
		}
		hrs, ok := c.Hours[dayStart.Weekday()]
		if !ok {
			cur = dayEnd
			continue
		}
		bhStart := dayStart.Add(time.Duration(hrs.StartSec) * time.Second)
		bhEnd := dayStart.Add(time.Duration(hrs.EndSec) * time.Second)
		if cur.Before(bhStart) {
			cur = bhStart
		}
		if cur.After(bhEnd) {
			cur = dayEnd
			continue
		}
		e := minTime(end, bhEnd)
		if e.After(cur) {
			total += e.Sub(cur)
		}
		cur = e
		if cur.Equal(bhEnd) {
			cur = dayEnd
		}
	}
	return total
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

