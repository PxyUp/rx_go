package rx_go

import (
	"sync"
)

type Observer[T any] struct {
	list chan T

	onComplete  func()
	onSubscribe func()
	onNext      func(value T)

	completed bool
	mutex     sync.Mutex
}

// ArrayObserver create observer from array
func ArrayObserver[T any](items ...T) *Observer[T] {
	obs := NewObserver[T]()

	go func() {
		for _, j := range items {
			obs.Next(j)
		}
		obs.Complete()
	}()
	return obs
}

// StaticObserver create static observer from one value
func StaticObserver[T any](value T) *Observer[T] {
	obs := NewObserver[T]()
	go func() {
		obs.Next(value)
		obs.Complete()
	}()
	return obs
}

func NewObserver[T any]() *Observer[T] {
	return &Observer[T]{
		list:        make(chan T),
		onComplete:  func() {},
		onSubscribe: func() {},
		onNext:      func(v T) {},
	}
}

func (o *Observer[T]) SetOnComplete(fn func()) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.onComplete = fn
}

func (o *Observer[T]) SetOnNext(fn func(v T)) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.onNext = fn
}

func (o *Observer[T]) SetOnSubscribe(fn func()) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.onSubscribe = fn
}

func (o *Observer[T]) Next(value T) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.completed {
		return
	}
	o.list <- value
	o.onNext(value)
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
