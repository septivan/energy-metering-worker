package anomaly

import (
	"fmt"
)

// Detector handles anomaly detection with configurable thresholds
type Detector struct {
	spikeThreshold            float64
	minDataPointsForDetection int
}

// NewDetector creates a new anomaly detector with the specified thresholds
func NewDetector(spikeThreshold float64, minDataPointsForDetection int) *Detector {
	return &Detector{
		spikeThreshold:            spikeThreshold,
		minDataPointsForDetection: minDataPointsForDetection,
	}
}

// DetectAnomaly checks if the value is anomalous based on historical data
func (d *Detector) DetectAnomaly(value float64, historicalValues []float64) (bool, string) {
	// Check for negative values
	if value < 0 {
		return true, "negative value"
	}

	// Need enough historical data for spike detection
	if len(historicalValues) < d.minDataPointsForDetection {
		return false, ""
	}

	// Calculate rolling average
	sum := 0.0
	for _, v := range historicalValues {
		sum += v
	}
	average := sum / float64(len(historicalValues))

	// Detect sudden spike (>threshold x rolling average)
	if average > 0 && value > d.spikeThreshold*average {
		return true, fmt.Sprintf("sudden spike detected: value %.2f exceeds %.1fx rolling average %.2f",
			value, d.spikeThreshold, average)
	}

	return false, ""
}
