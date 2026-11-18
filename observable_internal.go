package rx_go

import (
	"context"
	"sync"
)

type subscribeFunc[T any] func(context.Context, *Observer[T]) func()

// Observable is a lazy stream. Every subscription invokes subscribe anew.
type Observable[T any] struct{ subscribe subscribeFunc[T] }

// Operator transforms an Observable while preserving its element type.
type Operator[T any] func(*Observable[T]) *Observable[T]

// Subscription controls one independent execution of an Observable.
type Subscription struct {
	cancel context.CancelFunc
	done   <-chan struct{}
	once   sync.Once
}

func (s *Subscription) Cancel() {
	if s != nil {
		s.once.Do(s.cancel)
	}
}
func (s *Subscription) Done() <-chan struct{} {
	if s == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return s.done
}

func create[T any](fn subscribeFunc[T]) *Observable[T] { return &Observable[T]{subscribe: fn} }

// Create constructs a cold Observable. The producer should stop when ctx is
// cancelled and may return an additional idempotent teardown function.
func Create[T any](producer func(context.Context, *Observer[T]) func()) *Observable[T] {
	return create(producer)
}

func (o *Observable[T]) Pipe(operators ...Operator[T]) *Observable[T] {
	result := o
	for _, operator := range operators {
		result = operator(result)
	}
	return result
}

// SubscribeObserver exposes the complete Rx notification contract.
func (o *Observable[T]) SubscribeObserver(ctx context.Context, next func(T), onError func(error), complete func()) *Subscription {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	if next == nil {
		next = func(T) {}
	}
	done := make(chan struct{})
	var terminal sync.Once
	finish := func(fn func()) {
		terminal.Do(func() {
			if fn != nil {
				fn()
			}
			close(done)
			cancel()
		})
	}
	sink := newSink(func(v T) {
		select {
		case <-ctx.Done():
			return
		default:
			next(v)
		}
	}, func(err error) {
		finish(func() {
			if onError != nil {
				onError(err)
			}
		})
	}, func() { finish(complete) })
	teardown := o.subscribe(ctx, sink)
	if teardown == nil {
		teardown = func() {}
	}
	go func() { <-ctx.Done(); teardown(); finish(nil) }()
	return &Subscription{cancel: cancel, done: done}
}

// Subscribe returns a value channel and cancellation function for compatibility.
// Use SubscribeObserver when errors must be observed directly.
func (o *Observable[T]) Subscribe(ctxs ...context.Context) (chan T, func()) {
	parent := context.Background()
	if len(ctxs) > 0 && ctxs[0] != nil {
		parent = ctxs[0]
	}
	ctx, stop := context.WithCancel(parent)
	out := make(chan T)
	sub := o.SubscribeObserver(ctx, func(v T) {
		select {
		case out <- v:
		case <-ctx.Done():
		}
	}, nil, nil)
	go func() { <-sub.Done(); close(out) }()
	return out, func() { stop(); sub.Cancel() }
}
