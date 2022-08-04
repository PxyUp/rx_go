package rx_go

import (
	"context"
	"sync"
	"time"
)

// Repeat emit value multiple times
func Repeat[T any](times uint32) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			defer observer.Complete()
			for value := range obs.list {
				local := value
				count := times
				for count > 0 {
					observer.Next(local)
					count--
				}
			}

		}()
		return observer
	}
}

// Debounce emit value if in provided amount of time new value was not emitted
func Debounce[T any](duration time.Duration) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			var wg sync.WaitGroup
			defer func() {
				wg.Wait()
				observer.Complete()
			}()
			var timer *time.Timer
			for value := range obs.list {
				local := value
				if timer != nil {
					wg.Done()
					timer.Stop()
				}
				wg.Add(1)
				timer = time.AfterFunc(duration, func() {
					wg.Done()
					observer.Next(local)
					timer = nil
				})
			}
		}()
		return observer
	}
}

// Do execute some action on the emitting value
func Do[T any](fn func(value T)) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			defer observer.Complete()
			for value := range obs.list {
				local := value
				fn(local)
				observer.Next(local)
			}
		}()
		return observer
	}
}

// Until emit value until context not done, if ctx is done value will not emit
func Until[T any](ctx context.Context) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			defer observer.Complete()
			for {
				select {
				case <-ctx.Done():
					return
				case value, ok := <-obs.list:
					if !ok {
						return
					}
					observer.Next(value)
				}
			}
		}()
		return observer
	}
}

// DistinctWith compare prev value and next with comparator and emit if they not equal
func DistinctWith[T any](comp func(T, T) bool) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			var prev *T
			for value := range obs.list {
				local := value
				if prev == nil {
					prev = &local
					observer.Next(local)
				} else {
					if !comp(*prev, local) {
						prev = &local
						observer.Next(local)
					}
				}
			}
			observer.Complete()
		}()
		return observer
	}
}

// Distinct compare prev value with next value and emit if they not same
func Distinct[T comparable]() Operator[T] {
	return DistinctWith(func(a T, b T) bool {
		return a == b
	})
}

// Take return first N elements from the begining
func Take[T any](count int) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			defer observer.Complete()
			for value := range obs.list {
				if count > 0 {
					observer.Next(value)
					count--
				}
			}
		}()
		return observer
	}
}

// Delay emit value with some delay in between
func Delay[T any](delay time.Duration) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			for v := range obs.list {
				time.Sleep(delay)
				observer.Next(v)
			}
			observer.Complete()
		}()
		return observer
	}
}

// FirstOne return first emitted element from observable
func FirstOne[T any]() Operator[T] {
	return Take[T](1)
}

// LastOne return last element from observable
func LastOne[T any]() Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			var last *T
			for value := range obs.list {
				local := value
				last = &local
			}
			if last != nil {
				observer.Next(*last)
			}
			observer.Complete()
		}()
		return observer
	}
}

// Filter elements from the observable
func Filter[T any](flt func(value T) bool) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			for value := range obs.list {
				if flt(value) {
					observer.Next(value)
				}
			}
			observer.Complete()
		}()
		return observer
	}
}

// Map static mapping from observable
func Map[T any](mapper func(value T) T) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			for value := range obs.list {
				observer.Next(mapper(value))
			}
			observer.Complete()
		}()
		return observer
	}
}
