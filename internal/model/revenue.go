package model

import (
	"fmt"
	"time"
)

// RevenueStats contains information about the income and usage time of the table.
type RevenueStats struct {

	// Income is the amount of money earned from the table.
	Income int

	// UsageTime is the time during which the table was used.
	//
	// When calculating the income, we round table usage time up to the nearest hour.
	// UsageTime isn't rounded.
	UsageTime time.Duration
}

func (r RevenueStats) String() string {
	hours := int(r.UsageTime.Hours())
	minutes := int(r.UsageTime.Minutes()) % 60
	return fmt.Sprintf("%d %02d:%02d", r.Income, hours, minutes)
}
