package util

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestGetExponentialBackoff(t *testing.T) {
	equal(t, GetExponentialBackoff(-5), minBackoffDuration)
	equal(t, GetExponentialBackoff(0), minBackoffDuration)
	equal(t, GetExponentialBackoff(1), 500*time.Millisecond)
	equal(t, GetExponentialBackoff(2), 1000*time.Millisecond)
	equal(t, GetExponentialBackoff(3), 2000*time.Millisecond)
	equal(t, GetExponentialBackoff(4), 4000*time.Millisecond)
	equal(t, GetExponentialBackoff(5), maxBackoffDuration)
	equal(t, GetExponentialBackoff(6), maxBackoffDuration)
	equal(t, GetExponentialBackoff(math.MaxInt64), maxBackoffDuration)
}

func equal(t *testing.T, i, j interface{}) {
	if !reflect.DeepEqual(i, j) {
		t.Errorf("Expected %v, but got %v", j, i)
	}
}
