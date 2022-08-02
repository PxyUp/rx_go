package rx_go

type Observer[T any] struct {
	list chan T
}

func NewObserver[T any]() *Observer[T] {
	return &Observer[T]{
		list: make(chan T),
	}
}

func (o *Observer[T]) Next(value T) {
	o.list <- value
}

func (o *Observer[T]) Complete() {
	close(o.list)
}
