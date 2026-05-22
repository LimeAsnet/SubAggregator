package service

import (
	"fmt"
	"time"
)

const MonthYearLayout = "01-2006"

func ParseMonthYear(value string) (time.Time, error) {
	t, err := time.Parse(MonthYearLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrInvalidDate, err)
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func FormatMonthYearDate(t time.Time) string {
	return t.Format("2006-01-02")
}
