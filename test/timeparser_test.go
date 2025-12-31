package anomaly_test

import (
	"testing"
	"time"

	"github.com/septivank/energy-metering-worker/tools/timeparser"
)

func TestParseMeterTimestamp_Format1(t *testing.T) {
	dateStr := "29/12/2025 10:30:45"

	result, err := timeparser.ParseMeterTimestamp(dateStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	expected := time.Date(2025, 12, 29, 10, 30, 45, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseMeterTimestamp_Format2(t *testing.T) {
	dateStr := "29 10:30:45/12/2025"

	result, err := timeparser.ParseMeterTimestamp(dateStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	expected := time.Date(2025, 12, 29, 10, 30, 45, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseMeterTimestamp_RFC3339(t *testing.T) {
	dateStr := "2025-12-29T10:30:45Z"

	result, err := timeparser.ParseMeterTimestamp(dateStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	expected := time.Date(2025, 12, 29, 10, 30, 45, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestParseMeterTimestamp_Invalid(t *testing.T) {
	dateStr := "invalid-date-string"

	_, err := timeparser.ParseMeterTimestamp(dateStr)
	if err == nil {
		t.Error("Expected error for invalid timestamp")
	}
}

func TestIsWithinTolerance_WithinRange(t *testing.T) {
	readingTime := time.Date(2025, 12, 29, 10, 30, 0, 0, time.UTC)
	receivedTime := time.Date(2025, 12, 29, 10, 33, 0, 0, time.UTC) // 3 minutes later

	result := timeparser.IsWithinTolerance(readingTime, receivedTime, 5)
	if !result {
		t.Error("Expected timestamp to be within tolerance")
	}
}

func TestIsWithinTolerance_OutsideRange(t *testing.T) {
	readingTime := time.Date(2025, 12, 29, 10, 30, 0, 0, time.UTC)
	receivedTime := time.Date(2025, 12, 29, 10, 36, 0, 0, time.UTC) // 6 minutes later

	result := timeparser.IsWithinTolerance(readingTime, receivedTime, 5)
	if result {
		t.Error("Expected timestamp to be outside tolerance")
	}
}

func TestIsWithinTolerance_NegativeDifference(t *testing.T) {
	readingTime := time.Date(2025, 12, 29, 10, 35, 0, 0, time.UTC)
	receivedTime := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC) // 3 minutes before

	result := timeparser.IsWithinTolerance(readingTime, receivedTime, 5)
	if !result {
		t.Error("Expected timestamp to be within tolerance (negative difference)")
	}
}

func TestIsWithinTolerance_ExactBoundary(t *testing.T) {
	readingTime := time.Date(2025, 12, 29, 10, 30, 0, 0, time.UTC)
	receivedTime := time.Date(2025, 12, 29, 10, 35, 0, 0, time.UTC) // Exactly 5 minutes

	result := timeparser.IsWithinTolerance(readingTime, receivedTime, 5)
	if !result {
		t.Error("Expected timestamp at exact boundary to be within tolerance")
	}
}
