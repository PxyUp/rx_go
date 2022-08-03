package rx_go

import "time"

// IntervalObserver return Observer which produce value periodically
func IntervalObserver(interval time.Duration, startNow bool) *Observer[time.Time] {
	timer := time.NewTimer(interval)
	obs := NewObserver[time.Time]()

	go func() {
		if startNow {
			obs.Next(time.Now())
		}

		for v := range timer.C {
			obs.Next(v)
		}
		obs.Complete()
	}()

	obs.onComplete = func() {
		timer.Stop()
	}

	return obs
}
