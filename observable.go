package rx_go

type Observable[T any] struct {
	observer *Observer[T]
}

type Operator[T any] func(observer *Observer[T]) *Observer[T]

// New create new observable with predefined observer
func New[T any](observer *Observer[T]) *Observable[T] {
	return &Observable[T]{
		observer: observer,
	}
}

// From create new observable from static array
func From[T any](array ...T) *Observable[T] {
	obs := NewObserver[T]()
	go func() {
		for _, j := range array {
			obs.Next(j)
		}
		obs.Complete()
	}()

	return &Observable[T]{
		observer: obs,
	}
}

// MapTo create new observable with modified values
func MapTo[T any, Y any](o *Observable[T], mapper func(T) Y) *Observable[Y] {
	obs := NewObserver[Y]()
	go func() {
		for oldValue := range o.observer.list {
			obs.Next(mapper(oldValue))
		}
		obs.Complete()
	}()
	return New(obs)
}

func (o *Observable[T]) Pipe(operators ...Operator[T]) *Observable[T] {
	current := o.observer
	for _, op := range operators {
		current = op(current)
	}
	return New(current)
}

func (o *Observable[T]) Subscribe() chan T {
	t := make(chan T)
	go func() {
		defer close(t)
		for value := range o.observer.list {
			t <- value
		}
	}()
	return t
}
