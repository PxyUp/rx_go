package rx_go

import (
	"time"
)

// IntervalObserver return Observer which produce value periodically
func IntervalObserver(interval time.Duration, startNow bool) *Observer[time.Time] {
	ticker := time.NewTicker(interval)
	obs := NewObserver[time.Time]()

	obs.SetOnComplete(func() {
		ticker.Stop()
	})

	waiting := make(chan struct{})
	subscribed := false

	obs.onSubscribe = func() {
		if subscribed {
			return
		}

		subscribed = true
		close(waiting)
	}

	go func() {
		<-waiting

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
