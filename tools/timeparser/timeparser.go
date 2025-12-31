package timeparser

import (
	"fmt"
	"time"
)

// ParseMeterTimestamp attempts to parse meter timestamp with multiple formats
func ParseMeterTimestamp(dateStr string) (time.Time, error) {
	formats := []string{
		"02/01/2006 15:04:05", // DD/MM/YYYY HH:mm:ss
		"02 15:04:05/01/2006", // DD HH:mm:ss/MM/YYYY
		time.RFC3339,          // Standard RFC3339
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("failed to parse timestamp '%s': %w", dateStr, lastErr)
}

// IsWithinTolerance checks if the reading timestamp is within tolerance of received time
func IsWithinTolerance(readingTime, receivedTime time.Time, toleranceMinutes int) bool {
	diff := readingTime.Sub(receivedTime)
	if diff < 0 {
		diff = -diff
	}
	return diff <= time.Duration(toleranceMinutes)*time.Minute
}
