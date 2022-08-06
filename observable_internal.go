package rx_go

import (
	"context"
)

type Observable[T any] struct {
	observer *Observer[T]
}

type Operator[T any] func(observer *Observer[T]) *Observer[T]

func (o *Observable[T]) Pipe(operators ...Operator[T]) *Observable[T] {
	old := o.observer
	for _, op := range operators {

		copyOldOnCompleteFn := old.onComplete
		copyOldOnSubscribeFn := old.onSubscribe

		newObs := op(old)

		copyNewOnCompleteFn := newObs.onComplete
		copyNewOnSubscribeFn := newObs.onSubscribe

		newObs.SetOnComplete(func() {
			copyOldOnCompleteFn()
			copyNewOnCompleteFn()
		})

		newObs.SetOnSubscribe(func() {
			copyOldOnSubscribeFn()
			copyNewOnSubscribeFn()
		})

		old = newObs
	}
	return New(old)
}

// Subscribe - create channel for reading values and unsubscribe function
func (o *Observable[T]) Subscribe() (chan T, func()) {
	t := make(chan T)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		o.observer.Complete()
	}()

	go func() {
		if o.observer.onSubscribe != nil {
			o.observer.onSubscribe()
		}

		defer close(t)
		for {
			select {
			case <-ctx.Done():
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
