package domain

import (
	"fmt"
	"time"
)

var IST = mustLoad("Asia/Kolkata")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("IST", 5*3600+1800)
	}
	return loc
}

// BusinessDay is the shop-local calendar date (tokens restart at 01 every day).
// R1a simplification of FS-9.7: the day starts at local midnight, not at opening time.
func BusinessDay(now time.Time, loc *time.Location) string {
	return now.In(loc).Format("2006-01-02")
}

// FormatToken renders "A-07"; three digits after 99.
func FormatToken(letter string, n int) string {
	return fmt.Sprintf("%s-%02d", letter, n)
}

// WaitEstimate is the queue wait shown to customers (FSD §9.3).
type WaitEstimate struct {
	JobsAhead   int `json:"jobsAhead"`
	LowMinutes  int `json:"lowMinutes"`
	HighMinutes int `json:"highMinutes"`
}

// EstimateWait: each job ahead takes perJob minutes plus one minute per 20 pages,
// shared across the active counters, shown as a 5-minute range.
func EstimateWait(pagesAhead []int, perJob float64, counters int) WaitEstimate {
	if counters < 1 {
		counters = 1
	}
	if perJob <= 0 {
		perJob = 3
	}
	var mins float64
	for _, p := range pagesAhead {
		mins += perJob + float64(p)/20
	}
	mins /= float64(counters)
	w := WaitEstimate{JobsAhead: len(pagesAhead)}
	if len(pagesAhead) == 0 {
		return w
	}
	low := int(mins) / 5 * 5
	high := low + 5
	if low == 0 {
		low = 1
	}
	w.LowMinutes, w.HighMinutes = low, high
	return w
}
