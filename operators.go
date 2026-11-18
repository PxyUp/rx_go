package rx_go

import (
	"context"
	"errors"
	"sync"
	"time"
)

// These notification types live at package scope because Go 1.18 does not
// support type declarations inside generic functions.
type delayedEvent[T any] struct {
	value    T
	err      error
	complete bool
	at       time.Time
}

type debounceEvent[T any] struct {
	value    T
	err      error
	complete bool
}

func lift[T any](next func(T, *Observer[T]), complete func(*Observer[T])) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			return source.subscribe(ctx, newSink(func(v T) { next(v, dst) }, dst.Error, func() {
				if complete != nil {
					complete(dst)
				} else {
					dst.Complete()
				}
			}))
		})
	}
}

func Map[T any](mapper func(T) T) Operator[T] {
	return lift(func(v T, dst *Observer[T]) { dst.Next(mapper(v)) }, nil)
}
func Filter[T any](predicate func(T) bool) Operator[T] {
	return lift(func(v T, dst *Observer[T]) {
		if predicate(v) {
			dst.Next(v)
		}
	}, nil)
}
func Do[T any](action func(T)) Operator[T] {
	return lift(func(v T, dst *Observer[T]) { action(v); dst.Next(v) }, nil)
}
func StartWith[T any](values ...T) Operator[T] {
	return func(source *Observable[T]) *Observable[T] { return Concat(From(values...), source) }
}
func EndWith[T any](values ...T) Operator[T] {
	return func(source *Observable[T]) *Observable[T] { return Concat(source, From(values...)) }
}

func Skip[T any](count uint32) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			remaining := count
			return source.subscribe(ctx, newSink(func(v T) {
				if remaining > 0 {
					remaining--
					return
				}
				dst.Next(v)
			}, dst.Error, dst.Complete))
		})
	}
}

func Take[T any](count int) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			if count <= 0 {
				dst.Complete()
				return nil
			}
			innerCtx, cancel := context.WithCancel(ctx)
			remaining := count
			teardown := source.subscribe(innerCtx, newSink(func(v T) {
				if remaining <= 0 {
					return
				}
				remaining--
				dst.Next(v)
				if remaining == 0 {
					dst.Complete()
					cancel()
				}
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

func FirstOne[T any]() Operator[T] { return Take[T](1) }

func LastOne[T any]() Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			var value T
			found := false
			return source.subscribe(ctx, newSink(func(v T) { value, found = v, true }, dst.Error, func() {
				if found {
					dst.Next(value)
				}
				dst.Complete()
			}))
		})
	}
}

func Find[T any](predicate func(T) bool) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			innerCtx, cancel := context.WithCancel(ctx)
			found := false
			teardown := source.subscribe(innerCtx, newSink(func(v T) {
				if !found && predicate(v) {
					found = true
					dst.Next(v)
					dst.Complete()
					cancel()
				}
			}, dst.Error, func() {
				if !found {
					dst.Complete()
				}
			}))
			return func() {
				cancel()
				if teardown != nil {
					teardown()
				}
			}
		})
	}
}

func ElementAt[T any](index uint32) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			innerCtx, cancel := context.WithCancel(ctx)
			current := uint32(0)
			found := false
			teardown := source.subscribe(innerCtx, newSink(func(v T) {
				if current == index {
					found = true
					dst.Next(v)
					dst.Complete()
					cancel()
				}
				current++
			}, dst.Error, func() {
				if !found {
					dst.Error(errors.New("rx_go: element index out of range"))
				}
			}))
			return func() {
				cancel()
				if teardown != nil {
					teardown()
				}
			}
		})
	}
}

func Distinct[T comparable]() Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			seen := make(map[T]struct{})
			return source.subscribe(ctx, newSink(func(v T) {
				if _, ok := seen[v]; !ok {
					seen[v] = struct{}{}
					dst.Next(v)
				}
			}, dst.Error, dst.Complete))
		})
	}
}

func DistinctWith[T any](equal func(T, T) bool) Operator[T] { return DistinctUntilChangedWith(equal) }
func DistinctUntilChanged[T comparable]() Operator[T] {
	return DistinctUntilChangedWith(func(a, b T) bool { return a == b })
}
func DistinctUntilChangedWith[T any](equal func(T, T) bool) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			var previous T
			hasPrevious := false
			return source.subscribe(ctx, newSink(func(v T) {
				if !hasPrevious || !equal(previous, v) {
					previous, hasPrevious = v, true
					dst.Next(v)
				}
			}, dst.Error, dst.Complete))
		})
	}
}

// Scan emits every intermediate accumulator (with the element type as state).
func Scan[T any](reducer func(T, T) T, seed T) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			acc := seed
			return source.subscribe(ctx, newSink(func(v T) { acc = reducer(acc, v); dst.Next(acc) }, dst.Error, dst.Complete))
		})
	}
}

// Repeat resubscribes to the complete source sequence. times=0 completes empty.
func Repeat[T any](times uint32) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				for i := uint32(0); i < times; i++ {
					done := make(chan struct{})
					var once sync.Once
					source.subscribe(ctx, newSink(dst.Next, func(err error) { once.Do(func() { dst.Error(err); cancel(); close(done) }) }, func() { once.Do(func() { close(done) }) }))
					select {
					case <-ctx.Done():
						return
					case <-done:
					}
				}
				dst.Complete()
			}()
			return cancel
		})
	}
}

func Delay[T any](delay time.Duration) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			events := make(chan delayedEvent[T])
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case e := <-events:
						t := time.NewTimer(time.Until(e.at))
						select {
						case <-ctx.Done():
							t.Stop()
							return
						case <-t.C:
						}
						if e.err != nil {
							dst.Error(e.err)
							return
						}
						if e.complete {
							dst.Complete()
							return
						}
						dst.Next(e.value)
					}
				}
			}()
			emit := func(e delayedEvent[T]) {
				e.at = time.Now().Add(delay)
				select {
				case events <- e:
				case <-ctx.Done():
				}
			}
			return source.subscribe(ctx, newSink(func(v T) { emit(delayedEvent[T]{value: v}) }, func(err error) { emit(delayedEvent[T]{err: err}) }, func() { emit(delayedEvent[T]{complete: true}) }))
		})
	}
}

func InitialDelay[T any](delay time.Duration) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			var mu sync.Mutex
			var teardown func()
			t := time.AfterFunc(delay, func() { mu.Lock(); teardown = source.subscribe(ctx, dst); mu.Unlock() })
			return func() {
				t.Stop()
				cancel()
				mu.Lock()
				defer mu.Unlock()
				if teardown != nil {
					teardown()
				}
			}
		})
	}
}

func Debounce[T any](duration time.Duration) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			events := make(chan debounceEvent[T])
			go func() {
				var timer *time.Timer
				var timerC <-chan time.Time
				var latest T
				pending := false
				upstreamDone := false
				for {
					select {
					case <-ctx.Done():
						if timer != nil {
							timer.Stop()
						}
						return
					case e := <-events:
						if e.err != nil {
							dst.Error(e.err)
							return
						}
						if e.complete {
							upstreamDone = true
							if !pending {
								dst.Complete()
								return
							}
							continue
						}
						latest, pending = e.value, true
						if timer == nil {
							timer = time.NewTimer(duration)
						} else {
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
							timer.Reset(duration)
						}
						timerC = timer.C
					case <-timerC:
						if pending {
							dst.Next(latest)
							pending = false
						}
						timerC = nil
						if upstreamDone {
							dst.Complete()
							return
						}
					}
				}
			}()
			emit := func(e debounceEvent[T]) {
				select {
				case events <- e:
				case <-ctx.Done():
				}
			}
			return source.subscribe(ctx, newSink(func(v T) { emit(debounceEvent[T]{value: v}) }, func(err error) { emit(debounceEvent[T]{err: err}) }, func() { emit(debounceEvent[T]{complete: true}) }))
		})
	}
}

// Throttle emits the first value in each duration window.
func Throttle[T any](duration time.Duration) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			var mu sync.Mutex
			open := true
			return source.subscribe(ctx, newSink(func(v T) {
				mu.Lock()
				if !open {
					mu.Unlock()
					return
				}
				open = false
				mu.Unlock()
				dst.Next(v)
				time.AfterFunc(duration, func() { mu.Lock(); open = true; mu.Unlock() })
			}, dst.Error, dst.Complete))
		})
	}
}

func Finally[T any](action func()) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			var once sync.Once
			run := func() { once.Do(action) }
			teardown := source.subscribe(ctx, newSink(dst.Next, func(err error) { run(); dst.Error(err) }, func() { run(); dst.Complete() }))
			return func() {
				if teardown != nil {
					teardown()
				}
				run()
			}
		})
	}
}

func UntilCtx[T any](until context.Context) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				select {
				case <-until.Done():
					dst.Complete()
					cancel()
				case <-ctx.Done():
				}
			}()
			teardown := source.subscribe(ctx, dst)
			return func() {
				cancel()
				if teardown != nil {
					teardown()
				}
			}
		})
	}
}

func AfterCtx[T any](after context.Context) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-after.Done():
					source.subscribe(ctx, dst)
				}
			}()
			return cancel
		})
	}
}

func SkipUntilCtx[T any](until context.Context) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			return source.subscribe(ctx, newSink(func(v T) {
				select {
				case <-until.Done():
					dst.Next(v)
				default:
				}
			}, dst.Error, dst.Complete))
		})
	}
}

func SkipUntil[T, U any](trigger *Observable[U]) Operator[T] {
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			ctx, cancel := context.WithCancel(ctx)
			var mu sync.RWMutex
			open := false
			trigger.subscribe(ctx, newSink(func(U) { mu.Lock(); open = true; mu.Unlock() }, dst.Error, func() {}))
			teardown := source.subscribe(ctx, newSink(func(v T) {
				mu.RLock()
				pass := open
				mu.RUnlock()
				if pass {
					dst.Next(v)
				}
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
