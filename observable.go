package rx_go

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	Empty = EmptyOf[any]()
	Never = NeverOf[any]()
)

// New exposes an Observer as a hot multicast Observable.
func New[T any](observer *Observer[T]) *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() { return observer.subscribe(ctx, dst) })
}

func EmptyOf[T any]() *Observable[T] {
	return create(func(_ context.Context, dst *Observer[T]) func() { dst.Complete(); return nil })
}

func NeverOf[T any]() *Observable[T] {
	return create(func(_ context.Context, _ *Observer[T]) func() { return nil })
}

func Throw[T any](err error) *Observable[T] {
	return create(func(_ context.Context, dst *Observer[T]) func() { dst.Error(err); return nil })
}

func Defer[T any](factory func() *Observable[T]) *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		o := factory()
		if o == nil {
			dst.Error(errors.New("rx_go: Defer factory returned nil"))
			return nil
		}
		return o.subscribe(ctx, dst)
	})
}

func From[T any](items ...T) *Observable[T] {
	copyItems := append([]T(nil), items...)
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		go func() {
			for _, item := range copyItems {
				select {
				case <-ctx.Done():
					return
				default:
					dst.Next(item)
				}
			}
			dst.Complete()
		}()
		return nil
	})
}

func Of[T any](item T) *Observable[T] { return From(item) }

func FromChannel[T any](input <-chan T) *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case value, ok := <-input:
					if !ok {
						dst.Complete()
						return
					}
					dst.Next(value)
				}
			}
		}()
		return nil
	})
}

func NewInterval(period time.Duration, startNow bool) *Observable[time.Time] {
	return create(func(ctx context.Context, dst *Observer[time.Time]) func() {
		if period <= 0 {
			dst.Error(errors.New("rx_go: interval must be positive"))
			return nil
		}
		ticker := time.NewTicker(period)
		go func() {
			defer ticker.Stop()
			if startNow {
				dst.Next(time.Now())
			}
			for {
				select {
				case <-ctx.Done():
					return
				case value := <-ticker.C:
					dst.Next(value)
				}
			}
		}()
		return ticker.Stop
	})
}

func Timer(delay time.Duration) *Observable[time.Time] {
	return create(func(ctx context.Context, dst *Observer[time.Time]) func() {
		t := time.NewTimer(delay)
		go func() {
			defer t.Stop()
			select {
			case <-ctx.Done():
				return
			case value := <-t.C:
				dst.Next(value)
				dst.Complete()
			}
		}()
		return func() { t.Stop() }
	})
}

// NewHttp performs the request lazily for every subscription.
func NewHttp(client *http.Client, req *http.Request) (*Observable[[]byte], error) {
	if req == nil {
		return nil, errors.New("rx_go: nil HTTP request")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return create(func(ctx context.Context, dst *Observer[[]byte]) func() {
		request := req.Clone(ctx)
		go func() {
			resp, err := client.Do(request)
			if err != nil {
				dst.Error(err)
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				dst.Error(err)
				return
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				dst.Error(&HTTPError{StatusCode: resp.StatusCode, Body: body})
				return
			}
			dst.Next(body)
			dst.Complete()
		}()
		return nil
	}), nil
}

type HTTPError struct {
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string { return http.StatusText(e.StatusCode) }

func MapTo[T, R any](source *Observable[T], mapper func(T) R) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		return source.subscribe(ctx, newSink(func(v T) { dst.Next(mapper(v)) }, dst.Error, dst.Complete))
	})
}

// Reduce emits the final accumulator once when the source completes.
func Reduce[T, R any](source *Observable[T], reducer func(R, T) R, seed R) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		acc := seed
		return source.subscribe(ctx, newSink(func(v T) { acc = reducer(acc, v) }, dst.Error, func() { dst.Next(acc); dst.Complete() }))
	})
}

func ScanTo[T, R any](source *Observable[T], reducer func(R, T) R, seed R) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		acc := seed
		return source.subscribe(ctx, newSink(func(v T) { acc = reducer(acc, v); dst.Next(acc) }, dst.Error, dst.Complete))
	})
}

func Pairwise[T any](source *Observable[T]) *Observable[[2]T] {
	return create(func(ctx context.Context, dst *Observer[[2]T]) func() {
		var previous T
		hasPrevious := false
		return source.subscribe(ctx, newSink(func(v T) {
			if hasPrevious {
				dst.Next([2]T{previous, v})
			}
			previous, hasPrevious = v, true
		}, dst.Error, dst.Complete))
	})
}

func Merge[T any](sources ...*Observable[T]) *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		ctx, cancel := context.WithCancel(ctx)
		if len(sources) == 0 {
			dst.Complete()
			return cancel
		}
		var wg sync.WaitGroup
		var once sync.Once
		for _, source := range sources {
			if source == nil {
				once.Do(func() { dst.Error(errors.New("rx_go: nil source")); cancel() })
				continue
			}
			wg.Add(1)
			source.subscribe(ctx, newSink(dst.Next, func(err error) { once.Do(func() { dst.Error(err); cancel() }) }, wg.Done))
		}
		go func() {
			wg.Wait()
			select {
			case <-ctx.Done():
			default:
				dst.Complete()
			}
		}()
		return cancel
	})
}

// Concat subscribes to each source in order.
func Concat[T any](sources ...*Observable[T]) *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			for _, source := range sources {
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

func ToSlice[T any](source *Observable[T]) *Observable[[]T] {
	return create(func(ctx context.Context, dst *Observer[[]T]) func() {
		values := make([]T, 0)
		return source.subscribe(ctx, newSink(func(v T) { values = append(values, v) }, dst.Error, func() { dst.Next(values); dst.Complete() }))
	})
}

func Switch[T, R any](source *Observable[T], mapper func(T) *Observable[R]) *Observable[R] {
	return create(func(ctx context.Context, dst *Observer[R]) func() {
		ctx, cancel := context.WithCancel(ctx)
		var mu sync.Mutex
		var innerCancel func()
		outerDone := false
		generation := uint64(0)
		finishIfDone := func() {
			mu.Lock()
			done := outerDone && innerCancel == nil
			mu.Unlock()
			if done {
				dst.Complete()
			}
		}
		outerCancel := source.subscribe(ctx, newSink(func(v T) {
			mu.Lock()
			generation++
			current := generation
			if innerCancel != nil {
				innerCancel()
			}
			innerCtx, stop := context.WithCancel(ctx)
			innerCancel = stop
			mu.Unlock()
			inner := mapper(v)
			if inner == nil {
				dst.Error(errors.New("rx_go: Switch mapper returned nil"))
				cancel()
				return
			}
			inner.subscribe(innerCtx, newSink(dst.Next, func(err error) { dst.Error(err); cancel() }, func() {
				mu.Lock()
				if generation == current {
					innerCancel = nil
				}
				mu.Unlock()
				finishIfDone()
			}))
		}, func(err error) { dst.Error(err); cancel() }, func() { mu.Lock(); outerDone = true; mu.Unlock(); finishIfDone() }))
		return func() {
			if outerCancel != nil {
				outerCancel()
			}
			mu.Lock()
			if innerCancel != nil {
				innerCancel()
			}
			mu.Unlock()
			cancel()
		}
	})
}

func ForkJoin[T any](sources ...*Observable[T]) *Observable[[]T] {
	return create(func(ctx context.Context, dst *Observer[[]T]) func() {
		ctx, cancel := context.WithCancel(ctx)
		if len(sources) == 0 {
			dst.Complete()
			return cancel
		}
		values := make([]T, len(sources))
		seen := make([]bool, len(sources))
		var wg sync.WaitGroup
		var once sync.Once
		for i, source := range sources {
			wg.Add(1)
			index := i
			source.subscribe(ctx, newSink(func(v T) { values[index] = v; seen[index] = true }, func(err error) { once.Do(func() { dst.Error(err); cancel() }); wg.Done() }, wg.Done))
		}
		go func() {
			wg.Wait()
			if ctx.Err() != nil {
				return
			}
			for _, ok := range seen {
				if !ok {
					dst.Complete()
					return
				}
			}
			dst.Next(values)
			dst.Complete()
		}()
		return cancel
	})
}

// BroadCast is retained for compatibility. Prefer Share or an explicit Subject.
func BroadCast[T any](source *Observable[T], size int) []*Observable[T] {
	if size <= 0 {
		return nil
	}
	outputs := make([]*Observable[T], size)
	// Each branch is independently subscribable and receives the full sequence.
	// Use Share when a single upstream execution is required.
	for i := range outputs {
		outputs[i] = source
	}
	return outputs
}
