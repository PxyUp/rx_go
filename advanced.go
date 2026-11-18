package rx_go

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FlatMap subscribes to all projected streams concurrently.
func FlatMap[T, R any](source *Observable[T], mapper func(T) *Observable[R]) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		ctx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		var mu sync.Mutex
		outerDone := false
		complete := func() {
			mu.Lock()
			done := outerDone
			mu.Unlock()
			if done {
				wg.Wait()
				if ctx.Err() == nil {
					dst.Complete()
				}
			}
		}
		outer := source.subscribe(ctx, newSink(func(v T) {
			inner := mapper(v)
			if inner == nil {
				dst.Error(errors.New("rx_go: FlatMap mapper returned nil"))
				cancel()
				return
			}
			wg.Add(1)
			inner.subscribe(ctx, newSink(dst.Next, func(err error) { dst.Error(err); cancel() }, func() { wg.Done(); complete() }))
		}, func(err error) { dst.Error(err); cancel() }, func() { mu.Lock(); outerDone = true; mu.Unlock(); complete() }))
		return func() {
			if outer != nil {
				outer()
			}
			cancel()
		}
	})
}

// ConcatMap subscribes to projected streams sequentially.
func ConcatMap[T, R any](source *Observable[T], mapper func(T) *Observable[R]) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		ctx, cancel := context.WithCancel(ctx)
		queue := make(chan *Observable[R])
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case inner, ok := <-queue:
					if !ok {
						dst.Complete()
						return
					}
					done := make(chan struct{})
					var once sync.Once
					inner.subscribe(ctx, newSink(dst.Next, func(err error) { once.Do(func() { dst.Error(err); cancel(); close(done) }) }, func() { once.Do(func() { close(done) }) }))
					select {
					case <-ctx.Done():
						return
					case <-done:
					}
				}
			}
		}()
		source.subscribe(ctx, newSink(func(v T) {
			inner := mapper(v)
			if inner == nil {
				dst.Error(errors.New("rx_go: ConcatMap mapper returned nil"))
				cancel()
				return
			}
			select {
			case queue <- inner:
			case <-ctx.Done():
			}
		}, func(err error) { dst.Error(err); cancel() }, func() { close(queue) }))
		return cancel
	})
}

func Catch[T any](handler func(error) *Observable[T]) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			source.subscribe(ctx, newSink(dst.Next, func(err error) {
				replacement := handler(err)
				if replacement == nil {
					dst.Error(errors.New("rx_go: Catch handler returned nil"))
					return
				}
				replacement.subscribe(ctx, dst)
			}, dst.Complete))
			return cancel
		})
	}
}

// Retry resubscribes after errors. retries is the number of retries after the
// initial attempt; a negative value retries until cancelled.
func Retry[T any](retries int) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				attempt := 0
				for {
					done := make(chan struct{})
					var failure error
					var once sync.Once
					source.subscribe(ctx, newSink(dst.Next, func(err error) { failure = err; once.Do(func() { close(done) }) }, func() { once.Do(func() { close(done) }) }))
					select {
					case <-ctx.Done():
						return
					case <-done:
					}
					if failure == nil {
						dst.Complete()
						return
					}
					if retries >= 0 && attempt >= retries {
						dst.Error(failure)
						return
					}
					attempt++
				}
			}()
			return cancel
		})
	}
}

var ErrTimeout = errors.New("rx_go: observable timeout")

func Timeout[T any](duration time.Duration) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			reset := make(chan struct{}, 1)
			go func() {
				t := time.NewTimer(duration)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-reset:
						if !t.Stop() {
							select {
							case <-t.C:
							default:
							}
						}
						t.Reset(duration)
					case <-t.C:
						dst.Error(ErrTimeout)
						cancel()
						return
					}
				}
			}()
			teardown := source.subscribe(ctx, newSink(func(v T) {
				select {
				case reset <- struct{}{}:
				default:
				}
				dst.Next(v)
			}, dst.Error, dst.Complete))
			return func() {
				cancel()
				if teardown != nil {
					teardown()
				}
			}
		})
	}
}

func Count[T any](source *Observable[T]) *Observable[int] {
	return create(func(ctx context.Context, dst *Observer[int]) func() {
		count := 0
		return source.subscribe(ctx, newSink(func(T) { count++ }, dst.Error, func() { dst.Next(count); dst.Complete() }))
	})
}

type Pair[A, B any] struct {
	First  A
	Second B
}

func Zip[A, B any](left *Observable[A], right *Observable[B]) *Observable[Pair[A, B]] {
	return create(func(ctx context.Context, dst *Observer[Pair[A, B]]) func() {
		ctx, cancel := context.WithCancel(ctx)
		leftQ := make(chan A)
		rightQ := make(chan B)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case a, ok := <-leftQ:
					if !ok {
						dst.Complete()
						cancel()
						return
					}
					select {
					case <-ctx.Done():
						return
					case b, ok := <-rightQ:
						if !ok {
							dst.Complete()
							cancel()
							return
						}
						dst.Next(Pair[A, B]{First: a, Second: b})
					}
				}
			}
		}()
		left.subscribe(ctx, newSink(func(v A) {
			select {
			case leftQ <- v:
			case <-ctx.Done():
			}
		}, func(err error) { dst.Error(err); cancel() }, func() { close(leftQ) }))
		right.subscribe(ctx, newSink(func(v B) {
			select {
			case rightQ <- v:
			case <-ctx.Done():
			}
		}, func(err error) { dst.Error(err); cancel() }, func() { close(rightQ) }))
		return cancel
	})
}

func CombineLatest[A, B, R any](left *Observable[A], right *Observable[B], combine func(A, B) R) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		ctx, cancel := context.WithCancel(ctx)
		var mu sync.Mutex
		var a A
		var b B
		hasA, hasB := false, false
		completed := 0
		emit := func() {
			if hasA && hasB {
				dst.Next(combine(a, b))
			}
		}
		complete := func() {
			mu.Lock()
			completed++
			done := completed == 2
			mu.Unlock()
			if done {
				dst.Complete()
			}
		}
		left.subscribe(ctx, newSink(func(v A) { mu.Lock(); a, hasA = v, true; emit(); mu.Unlock() }, func(err error) { dst.Error(err); cancel() }, complete))
		right.subscribe(ctx, newSink(func(v B) { mu.Lock(); b, hasB = v, true; emit(); mu.Unlock() }, func(err error) { dst.Error(err); cancel() }, complete))
		return cancel
	})
}
