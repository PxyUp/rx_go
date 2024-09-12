package rx_go

import (
	"net/http"
	"sync"
	"time"
)

var (
	// Empty -  create new Observer which just completed
	Empty = New(func() *Observer[any] {
		o := NewObserver[any]()
		go func() {
			o.Complete()
		}()
		return o
	}())

	// Never -  create new Observer which never emit
	Never = New(NewObserver[any]())
)

// New create new observable with predefined observer
func New[T any](observer *Observer[T]) *Observable[T] {
	return &Observable[T]{
		observer: observer,
	}
}

// FromChannel create new observable from readable channel
func FromChannel[T any](ch <-chan T) *Observable[T] {
	return New[T](ChannelObserver(ch))
}

// Of create static observable from one value
func Of[T any](item T) *Observable[T] {
	return New(StaticObserver(item))
}

// Concat create static observable witch emit single array of all values
func Concat[T any](o *Observable[T]) *Observable[[]T] {
	obs := NewObserver[[]T]()

	ch, cancel := o.Subscribe()
	obs.SetOnComplete(func() {
		cancel()
	})

	var res []T
	go func() {
		for v := range ch {
			res = append(res, v)
		}
		obs.Next(res)
		obs.Complete()
	}()

	return New(obs)
}

// From create new observable from static array
func From[T any](array ...T) *Observable[T] {
	return New(ArrayObserver(array...))
}

// NewInterval return Observable from IntervalObserver observer
func NewInterval(duration time.Duration, startNow bool) *Observable[time.Time] {
	return New[time.Time](IntervalObserver(duration, startNow))
}

// NewHttp - return Observable from HttpObserver
func NewHttp(client *http.Client, req *http.Request) (*Observable[[]byte], error) {
	obs, err := HttpObserver(client, req)
	if err != nil {
		return nil, err
	}
	return New[[]byte](obs), nil
}

// Switch change stream for each value
func Switch[T any, Y any](o *Observable[T], mapper func(T) *Observable[Y]) *Observable[Y] {
	obs := NewObserver[Y]()
	var cancelFns []func()

	obs.SetOnComplete(func() {
		for _, fn := range cancelFns {
			fn()
		}
	})

	go func() {
		topCh, cancelTop := o.Subscribe()
		cancelFns = append(cancelFns, cancelTop)

		for oldValue := range topCh {
			localValue := oldValue
			ch, cancel := mapper(localValue).Subscribe()
			cancelFns = append(cancelFns, cancel)
			for v := range ch {
				obs.Next(v)
			}
		}

		obs.Complete()
	}()
	return New(obs)
}

// ForkJoin - wait for Observables to complete and then combine last values they emitted; complete immediately if an empty array is passed.
func ForkJoin[T any](obss ...*Observable[T]) *Observable[[]T] {
	obs := NewObserver[[]T]()
	resp := make([]T, len(obss))
	clean := make(chan struct{})
	cleanFns := make([]func(), len(obss))
	obs.SetOnComplete(func() {
		close(clean)
		for _, v := range cleanFns {
			v()
		}
	})

	var wg sync.WaitGroup
	for i, o := range obss {
		wg.Add(1)
		go func(index int, ob *Observable[T]) {
			defer wg.Done()
			ch, cancel := ob.Pipe(LastOne[T]()).Subscribe()
			cleanFns[index] = cancel

			for {
				select {
				case <-clean:
					return
				default:
					value, ok := <-ch
					if !ok {
						return
					}
					resp[index] = value
				}
			}
		}(i, o)
	}

	go func() {
		wg.Wait()
		obs.Next(resp)
		obs.Complete()
	}()

	return New(obs)
}

// Pairwise - groups pairs of consecutive emissions together and emits them as an array of two values.
func Pairwise[T any](o *Observable[T]) *Observable[[2]T] {
	obs := NewObserver[[2]T]()

	go func() {
		defer obs.Complete()
		ch, cancel := o.Subscribe()
		obs.SetOnComplete(func() {
			cancel()
		})

		var prev *T
		for value := range ch {
			local := value
			if prev != nil {
				obs.Next([2]T{*prev, local})
			}
			prev = &local
		}
	}()

	return New(obs)
}

// Reduce - create new observable which return accumulation value from all previous emitted items
func Reduce[T any, Y any](o *Observable[T], mapper func(Y, T) Y, initValue Y) *Observable[Y] {
	obs := NewObserver[Y]()

	go func() {
		defer obs.Complete()
		ch, cancel := o.Subscribe()
		obs.SetOnComplete(func() {
			cancel()
		})
		iValue := initValue
		for value := range ch {
			iValue = mapper(iValue, value)
			obs.Next(iValue)
		}
	}()

	return New(obs)
}

// MapTo create new observable with modified values
func MapTo[T any, Y any](o *Observable[T], mapper func(T) Y) *Observable[Y] {
	obs := NewObserver[Y]()
	go func() {
		defer obs.Complete()
		ch, cancel := o.Subscribe()
		obs.SetOnComplete(func() {
			cancel()
		})

		for value := range ch {
			obs.Next(mapper(value))
		}
	}()

	return New(obs)
}

// BroadCast splitting output from the observable to many outputs
func BroadCast[T any](obs *Observable[T], size int) []*Observable[T] {
	obss := make([]*Observable[T], size)
	oo := make([]*Observer[T], size)
	for i := 0; i < size; i++ {
		oo[i] = NewObserver[T]()
		obss[i] = New(oo[i])
	}
	go func() {
		ch, cancel := obs.Subscribe()

		defer func() {
			for i := 0; i < size; i++ {
				go func(lI int) {
					oo[lI].Complete()
				}(i)
			}
		}()

		for i := 0; i < size; i++ {
			oo[i].SetOnComplete(func() {
				cancel()
			})
		}

		for value := range ch {
			var wg sync.WaitGroup
			wg.Add(size)
			for i := 0; i < size; i++ {
				go func(lI int, lValue T) {
					defer wg.Done()
					oo[lI].Next(lValue)
				}(i, value)
			}
			wg.Wait()
		}
	}()

	return obss
}

// Merge merging multi observables with same type into single one
func Merge[T any](obss ...*Observable[T]) *Observable[T] {
	observer := NewObserver[T]()
	cleanFns := make([]func(), len(obss))
	clean := make(chan struct{})
	observer.SetOnComplete(func() {
		close(clean)
		for _, v := range cleanFns {
			v()
		}
	})

	var wg sync.WaitGroup
	for i, o := range obss {
		wg.Add(1)
		go func(index int, ob *Observable[T]) {
			defer wg.Done()
			ch, cancel := ob.Subscribe()
			cleanFns[index] = cancel

			for {
				select {
				case <-clean:
					return
				default:
					value, ok := <-ch
					if !ok {
						return
					}
					observer.Next(value)
				}
			}
		}(i, o)
	}

	go func() {
		wg.Wait()
		observer.Complete()
	}()

	return New(observer)
}
