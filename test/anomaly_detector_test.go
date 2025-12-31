package anomaly_test

import (
	"testing"

	"github.com/septivank/energy-metering-worker/internal/anomaly"
)

const (
	testSpikeThreshold            = 3.0
	testMinDataPointsForDetection = 3
)

func TestDetectAnomaly_NegativeValue(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	isAnomaly, reason := detector.DetectAnomaly(-10.5, []float64{100, 105, 98})

	if !isAnomaly {
		t.Error("Expected anomaly for negative value")
	}

	if reason != "negative value" {
		t.Errorf("Expected reason 'negative value', got '%s'", reason)
	}
}

func TestDetectAnomaly_SuddenSpike(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	historical := []float64{100, 105, 98, 102, 99}
	value := 350.0 // More than 3x the average (~100)

	isAnomaly, reason := detector.DetectAnomaly(value, historical)

	if !isAnomaly {
		t.Error("Expected anomaly for sudden spike")
	}

	if reason == "" {
		t.Error("Expected reason for spike anomaly")
	}
}

func TestDetectAnomaly_NormalValue(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	historical := []float64{100, 105, 98, 102, 99}
	value := 103.0

	isAnomaly, reason := detector.DetectAnomaly(value, historical)

	if isAnomaly {
		t.Errorf("Expected no anomaly, but got: %s", reason)
	}
}

func TestDetectAnomaly_InsufficientData(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	historical := []float64{100, 105} // Less than MinDataPointsForDetection
	value := 300.0

	isAnomaly, _ := detector.DetectAnomaly(value, historical)

	// Should not detect spike with insufficient data (but still checks negative)
	if isAnomaly {
		t.Error("Should not detect spike with insufficient historical data")
	}
}

func TestDetectAnomaly_EmptyHistorical(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	historical := []float64{}
	value := 100.0

	isAnomaly, _ := detector.DetectAnomaly(value, historical)

	if isAnomaly {
		t.Error("Expected no anomaly with empty historical data and positive value")
	}
}

func TestDetectAnomaly_ZeroAverage(t *testing.T) {
	detector := anomaly.NewDetector(testSpikeThreshold, testMinDataPointsForDetection)

	historical := []float64{0, 0, 0}
	value := 100.0

	isAnomaly, _ := detector.DetectAnomaly(value, historical)

	// Should not trigger spike detection when average is 0
	if isAnomaly {
		t.Error("Should not detect spike when historical average is 0")
	}
}
