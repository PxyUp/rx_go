package rx_go

import "sync"

type Observer[T any] struct {
	list       chan T
	onComplete func()
	completed  bool
	mutex      sync.Mutex
}

func NewObserver[T any]() *Observer[T] {
	return &Observer[T]{
		list: make(chan T),
	}
}

func (o *Observer[T]) Next(value T) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.completed {
		return
	}
	o.list <- value
}

func (o *Observer[T]) Complete() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.completed {
		return
	}
	o.completed = true
	if o.onComplete != nil {
		o.onComplete()
	}
	close(o.list)
}
