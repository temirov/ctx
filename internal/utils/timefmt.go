package utils

import (
	"time"
)

const timestampLayout = "2006-01-02 15:04"

// FormatTimestamp returns the provided time formatted using the local time zone
// and a layout that includes date and minutes (locale-sensitive via system TZ).
func FormatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.In(time.Local).Format(timestampLayout)
}
