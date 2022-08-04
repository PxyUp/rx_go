package rx_go

import (
	"net/http"
	"sync"
	"time"
)

// New create new observable with predefined observer
func New[T any](observer *Observer[T]) *Observable[T] {
	return &Observable[T]{
		observer: observer,
	}
}

// From create new observable from static array
func From[T any](array ...T) *Observable[T] {
	obs := NewObserver[T]()
	go func() {
		for _, j := range array {
			obs.Next(j)
		}
		obs.Complete()
	}()

	return &Observable[T]{
		observer: obs,
	}
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

// MapTo create new observable with modified values
func MapTo[T any, Y any](o *Observable[T], mapper func(T) Y) *Observable[Y] {
	obs := NewObserver[Y]()
	go func() {
		for oldValue := range o.observer.list {
			obs.Next(mapper(oldValue))
		}
		obs.Complete()
	}()
	return New(obs)
}

// Merge merging multi observers with same type into single one
func Merge[T any](obss ...*Observable[T]) *Observable[T] {
	observer := NewObserver[T]()

	cleanFns := make([]func(), len(obss))
	clean := make(chan struct{})
	var wg sync.WaitGroup
	for index, o := range obss {
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
		}(index, o)
	}

	observer.onComplete = func() {
		close(clean)
		for _, v := range cleanFns {
			v()
		}
	}

	go func() {
		wg.Wait()
		observer.Complete()
	}()

	return New(observer)
}
