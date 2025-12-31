package validator

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/septivank/energy-metering-worker/tools/timeparser"
)

// ValidationResult holds validation outcome
type ValidationResult struct {
	IsValid       bool
	AnomalyReason string
}

// MetricData represents a single metric reading
type MetricData struct {
	Date string
	Data string
	Name string
}

// Validator handles metric validation with configurable parameters
type Validator struct {
	timestampToleranceMinutes int
}

// NewValidator creates a new validator with the specified tolerance
func NewValidator(timestampToleranceMinutes int) *Validator {
	return &Validator{
		timestampToleranceMinutes: timestampToleranceMinutes,
	}
}

// ValidateMetricData validates a single metric reading
func (v *Validator) ValidateMetricData(metric MetricData, receivedAt time.Time) (float64, time.Time, ValidationResult) {
	result := ValidationResult{IsValid: true}

	// Validate metric name
	if metric.Name == "" {
		result.IsValid = false
		result.AnomalyReason = "empty metric name"
		return 0, time.Time{}, result
	}

	// Parse and validate metric value
	// Strip square brackets if present
	dataValue := strings.Trim(metric.Data, "[]")
	value, err := strconv.ParseFloat(dataValue, 64)
	if err != nil {
		result.IsValid = false
		result.AnomalyReason = fmt.Sprintf("invalid metric value: %v", err)
		return 0, time.Time{}, result
	}

	if value < 0 {
		result.IsValid = false
		result.AnomalyReason = "negative value detected"
		return value, time.Time{}, result
	}

	// Parse timestamp
	readingTime, err := timeparser.ParseMeterTimestamp(metric.Date)
	if err != nil {
		result.IsValid = false
		result.AnomalyReason = fmt.Sprintf("invalid timestamp format: %v", err)
		return value, time.Time{}, result
	}

	// Validate timestamp tolerance
	if !timeparser.IsWithinTolerance(readingTime, receivedAt, v.timestampToleranceMinutes) {
		result.IsValid = false
		result.AnomalyReason = fmt.Sprintf("timestamp outside tolerance window (±%d minutes)", v.timestampToleranceMinutes)
		return value, readingTime, result
	}

	return value, readingTime, result
}
