package rx_go

// ChannelObserver create http Observer from readable channel
func ChannelObserver[T any](ch <-chan T) *Observer[T] {
	obs := NewObserver[T]()

	go func() {
		for v := range ch {
			obs.Next(v)
		}
		obs.Complete()
	}()

	return obs
}
