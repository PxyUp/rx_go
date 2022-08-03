package rx_go

import (
	"time"
)

// IntervalObserver return Observer which produce value periodically
func IntervalObserver(interval time.Duration, startNow bool) *Observer[time.Time] {
	ticker := time.NewTicker(interval)
	obs := NewObserver[time.Time]()

	obs.onComplete = func() {
		ticker.Stop()
	}

	go func() {
		if startNow {
			obs.Next(time.Now())
		}

		for v := range ticker.C {
			obs.Next(v)
		}
		obs.Complete()
	}()

	return obs
}
