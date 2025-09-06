package model

import "time"

type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

type DateRange struct {
	Start time.Time
	End   time.Time
}
