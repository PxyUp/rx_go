package rx_go

import (
	"context"
	"sync"
)

// Observer receives the three kinds of Rx notifications. It is safe for
// concurrent use and accepts exactly one terminal notification.
//
// An Observer created with NewObserver is also a hot multicast source and can
// be passed to New. Observers created internally with newSink only forward to
// callbacks.
type Observer[T any] struct {
	mu          sync.Mutex
	terminated  bool
	onNext      func(T)
	onError     func(error)
	onComplete  func()
	onSubscribe func()
	subscribers map[uint64]*Observer[T]
	nextID      uint64
}

func newSink[T any](next func(T), onError func(error), complete func()) *Observer[T] {
	if next == nil {
		next = func(T) {}
	}
	if onError == nil {
		onError = func(error) {}
	}
	if complete == nil {
		complete = func() {}
	}
	return &Observer[T]{onNext: next, onError: onError, onComplete: complete}
}

// NewObserver creates a hot multicast observer suitable for manually emitting
// values and for use with New.
func NewObserver[T any]() *Observer[T] {
	return &Observer[T]{
		onNext:      func(T) {},
		onError:     func(error) {},
		onComplete:  func() {},
		onSubscribe: func() {},
		subscribers: make(map[uint64]*Observer[T]),
	}
}

func (o *Observer[T]) SetOnComplete(fn func())   { o.mu.Lock(); o.onComplete = fn; o.mu.Unlock() }
func (o *Observer[T]) SetOnError(fn func(error)) { o.mu.Lock(); o.onError = fn; o.mu.Unlock() }
func (o *Observer[T]) SetOnNext(fn func(T))      { o.mu.Lock(); o.onNext = fn; o.mu.Unlock() }
func (o *Observer[T]) SetOnSubscribe(fn func())  { o.mu.Lock(); o.onSubscribe = fn; o.mu.Unlock() }

func (o *Observer[T]) Next(value T) {
	o.mu.Lock()
	if o.terminated {
		o.mu.Unlock()
		return
	}
	fn := o.onNext
	subs := snapshot(o.subscribers)
	o.mu.Unlock()
	if fn != nil {
		fn(value)
	}
	for _, sub := range subs {
		sub.Next(value)
	}
}

func (o *Observer[T]) Error(err error) {
	if err == nil {
		return
	}
	o.mu.Lock()
	if o.terminated {
		o.mu.Unlock()
		return
	}
	o.terminated = true
	fn := o.onError
	subs := snapshot(o.subscribers)
	o.subscribers = nil
	o.mu.Unlock()
	if fn != nil {
		fn(err)
	}
	for _, sub := range subs {
		sub.Error(err)
	}
}

func (o *Observer[T]) Complete() {
	o.mu.Lock()
	if o.terminated {
		o.mu.Unlock()
		return
	}
	o.terminated = true
	fn := o.onComplete
	subs := snapshot(o.subscribers)
	o.subscribers = nil
	o.mu.Unlock()
	if fn != nil {
		fn()
	}
	for _, sub := range subs {
		sub.Complete()
	}
}

func snapshot[T any](in map[uint64]*Observer[T]) []*Observer[T] {
	if len(in) == 0 {
		return nil
	}
	out := make([]*Observer[T], 0, len(in))
	for _, sub := range in {
		out = append(out, sub)
	}
	return out
}

func (o *Observer[T]) subscribe(ctx context.Context, dst *Observer[T]) func() {
	o.mu.Lock()
	if o.terminated {
		o.mu.Unlock()
		dst.Complete()
		return func() {}
	}
	id := o.nextID
	o.nextID++
	o.subscribers[id] = dst
	hook := o.onSubscribe
	o.mu.Unlock()
	if hook != nil {
		hook()
	}
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			o.mu.Lock()
			delete(o.subscribers, id)
			o.mu.Unlock()
		})
	}
	go func() { <-ctx.Done(); cancel() }()
	return cancel
}

// ArrayObserver returns a hot observer which emits the supplied values.
// Prefer From when repeatable, cold behavior is desired.
func ArrayObserver[T any](items ...T) *Observer[T] {
	o := NewObserver[T]()
	go func() {
		for _, item := range items {
			o.Next(item)
		}
		o.Complete()
	}()
	return o
}

func StaticObserver[T any](value T) *Observer[T] { return ArrayObserver(value) }
