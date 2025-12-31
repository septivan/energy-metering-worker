package anomaly_test

import (
	"testing"
	"time"

	"github.com/septivank/energy-metering-worker/internal/validator"
)

const testTimestampToleranceMinutes = 5

func TestValidateMetricData_ValidData(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29/12/2025 10:30:00",
		Data: "245.5",
		Name: "power_consumption",
	}

	receivedAt := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC)

	value, timestamp, result := v.ValidateMetricData(metric, receivedAt)

	if !result.IsValid {
		t.Errorf("Expected valid result, got invalid: %s", result.AnomalyReason)
	}

	if value != 245.5 {
		t.Errorf("Expected value 245.5, got %f", value)
	}

	expectedTime := time.Date(2025, 12, 29, 10, 30, 0, 0, time.UTC)
	if !timestamp.Equal(expectedTime) {
		t.Errorf("Expected timestamp %v, got %v", expectedTime, timestamp)
	}
}

func TestValidateMetricData_NegativeValue(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29/12/2025 10:30:00",
		Data: "-10.5",
		Name: "power_consumption",
	}

	receivedAt := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC)

	_, _, result := v.ValidateMetricData(metric, receivedAt)

	if result.IsValid {
		t.Error("Expected invalid result for negative value")
	}

	if result.AnomalyReason != "negative value detected" {
		t.Errorf("Expected 'negative value detected', got '%s'", result.AnomalyReason)
	}
}

func TestValidateMetricData_EmptyName(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29/12/2025 10:30:00",
		Data: "245.5",
		Name: "",
	}

	receivedAt := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC)

	_, _, result := v.ValidateMetricData(metric, receivedAt)

	if result.IsValid {
		t.Error("Expected invalid result for empty name")
	}

	if result.AnomalyReason != "empty metric name" {
		t.Errorf("Expected 'empty metric name', got '%s'", result.AnomalyReason)
	}
}

func TestValidateMetricData_InvalidValue(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29/12/2025 10:30:00",
		Data: "not-a-number",
		Name: "power_consumption",
	}

	receivedAt := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC)

	_, _, result := v.ValidateMetricData(metric, receivedAt)

	if result.IsValid {
		t.Error("Expected invalid result for invalid value")
	}
}

func TestValidateMetricData_OutsideTolerance(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29/12/2025 10:00:00",
		Data: "245.5",
		Name: "power_consumption",
	}

	// Received 10 minutes later (outside ±5 minute tolerance)
	receivedAt := time.Date(2025, 12, 29, 10, 10, 1, 0, time.UTC)

	_, _, result := v.ValidateMetricData(metric, receivedAt)

	if result.IsValid {
		t.Error("Expected invalid result for timestamp outside tolerance")
	}
}

func TestValidateMetricData_AlternativeFormat(t *testing.T) {
	v := validator.NewValidator(testTimestampToleranceMinutes)

	metric := validator.MetricData{
		Date: "29 10:30:00/12/2025",
		Data: "245.5",
		Name: "power_consumption",
	}

	receivedAt := time.Date(2025, 12, 29, 10, 32, 0, 0, time.UTC)

	value, _, result := v.ValidateMetricData(metric, receivedAt)

	if !result.IsValid {
		t.Errorf("Expected valid result, got invalid: %s", result.AnomalyReason)
	}

	if value != 245.5 {
		t.Errorf("Expected value 245.5, got %f", value)
	}
}
