package rx_go

import "time"

func Take[T any](count int) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			for value := range obs.list {
				if count > 0 {
					observer.Next(value)
					count--
				}
			}
			observer.Complete()
		}()
		return observer
	}
}

func Delay[T any](delay time.Duration) Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			for v := range obs.list {
				observer.Next(v)
				time.Sleep(delay)
			}
			observer.Complete()
		}()
		return observer
	}
}

func First[T any]() Operator[T] {
	return func(obs *Observer[T]) *Observer[T] {
		observer := NewObserver[T]()
		go func() {
			var first *T
			for value := range obs.list {
				local := value
				if first == nil {
					first = &local
				}
			}
			if first != nil {
				observer.Next(*first)
			}
			observer.Complete()
		}()
		return observer
	}
}

func Last[T any]() Operator[T] {
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
