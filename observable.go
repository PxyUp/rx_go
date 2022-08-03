package rx_go

import (
	"context"
	"net/http"
	"time"
)

type Observable[T any] struct {
	observer *Observer[T]
}

type Operator[T any] func(observer *Observer[T]) *Observer[T]

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

func (o *Observable[T]) Pipe(operators ...Operator[T]) *Observable[T] {
	current := o.observer
	for _, op := range operators {
		current = op(current)
	}
	return New(current)
}

// Subscribe - create channel for reading values and unsubscribe function
func (o *Observable[T]) Subscribe() (chan T, func()) {
	t := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(t)
		for {
			select {
			case <-ctx.Done():
				o.observer.Complete()
				return
			case value, ok := <-o.observer.list:
				if !ok {
					return
				}
				t <- value
			}
		}
	}()
	return t, cancel
}
