package model

import (
	"time"
)

type TimeInterval struct {

	// Start is the time when the computer club opens.
	Start time.Time

	// End is the time when the computer club closes.
	End time.Time
}

func (ti *TimeInterval) In(t time.Time) bool {
	return ti.Start.Before(t) && ti.End.After(t)
}

func NewTimeInterval(start, end time.Time) *TimeInterval {
	if start.After(end) {
		end = end.AddDate(0, 0, 1)
	}

	return &TimeInterval{Start: start, End: end}
}

// CoreData is the main data that characterizes the computer club.
//
// Used in the event processing, calculating revenue, validating events.
type CoreData struct {

	// TablesCount is the number of tables in the computer club.
	TablesCount int

	// PricePerHour is the price per hour of using playing on a table.
	//
	// If player plays less than an hour, he still pays for an hour.
	PricePerHour int

	// WorkingTime is the time interval when the computer club is opens and closes.
	WorkingTime *TimeInterval
}

func NewCoreData(tablesCount, pricePerHour int, workingTime *TimeInterval) *CoreData {
	return &CoreData{
		TablesCount:  tablesCount,
		PricePerHour: pricePerHour,
		WorkingTime:  workingTime,
	}
}
