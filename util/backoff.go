package util

import (
	"math"
	"time"
)

const (
	minBackoffDuration = 250 * time.Millisecond
	maxBackoffDuration = 5 * time.Second
	base               = 2.0
)

// GetExponentialBackoff returns a Duration that increases exponentially with
// the number of attempts.
func GetExponentialBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return minBackoffDuration
	}
	durationf := float64(minBackoffDuration) * math.Pow(base, float64(attempt))
	if durationf > math.MaxInt64 {
		return maxBackoffDuration
	}
	duration := time.Duration(durationf)
	if duration > maxBackoffDuration {
		return maxBackoffDuration
	}
	return duration
}
