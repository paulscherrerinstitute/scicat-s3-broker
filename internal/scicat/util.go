package scicat

import "time"

// minTime returns min of two times, considering zero as a sentinel max time
func minTime(t1, t2 time.Time) time.Time {
	if t1.IsZero() {
		return t2
	}
	if t2.IsZero() {
		return t1
	}
	if t1.Before(t2) {
		return t1
	}
	return t2
}
