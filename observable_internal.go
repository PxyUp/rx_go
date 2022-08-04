package rx_go

import "context"

type Observable[T any] struct {
	observer *Observer[T]
}

type Operator[T any] func(observer *Observer[T]) *Observer[T]

func (o *Observable[T]) Pipe(operators ...Operator[T]) *Observable[T] {
	old := o.observer
	for _, op := range operators {
		copyOldCompleteFn := old.onComplete
		newObs := op(old)
		copyNewCompleteFn := newObs.onComplete
		newObs.onComplete = func() {
			copyOldCompleteFn()
			copyNewCompleteFn()
		}
		old = newObs
	}
	return New(old)
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
