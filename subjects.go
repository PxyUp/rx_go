package rx_go

import (
	"context"
	"sync"
)

// Subject is a hot multicast source and observer.
type Subject[T any] struct{ observer *Observer[T] }

func NewSubject[T any]() *Subject[T]             { return &Subject[T]{observer: NewObserver[T]()} }
func (s *Subject[T]) Next(v T)                   { s.observer.Next(v) }
func (s *Subject[T]) Error(err error)            { s.observer.Error(err) }
func (s *Subject[T]) Complete()                  { s.observer.Complete() }
func (s *Subject[T]) Observable() *Observable[T] { return New(s.observer) }

// BehaviorSubject replays its current value to each new subscriber.
type BehaviorSubject[T any] struct {
	mu     sync.RWMutex
	value  T
	replay *ReplaySubject[T]
}

// ReplaySubject replays up to capacity recent values to new subscribers.
type ReplaySubject[T any] struct {
	mu       sync.Mutex
	capacity int
	values   []T
	terminal bool
	err      error
	subs     map[uint64]*Observer[T]
	nextID   uint64
}

func NewReplaySubject[T any](capacity int) *ReplaySubject[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &ReplaySubject[T]{capacity: capacity, subs: make(map[uint64]*Observer[T])}
}

func (s *ReplaySubject[T]) Next(v T) {
	s.mu.Lock()
	if s.terminal {
		s.mu.Unlock()
		return
	}
	if s.capacity > 0 {
		s.values = append(s.values, v)
		if len(s.values) > s.capacity {
			s.values = append([]T(nil), s.values[len(s.values)-s.capacity:]...)
		}
	}
	subs := snapshot(s.subs)
	s.mu.Unlock()
	for _, sub := range subs {
		sub.Next(v)
	}
}
func (s *ReplaySubject[T]) Error(err error) { s.terminate(err) }
func (s *ReplaySubject[T]) Complete()       { s.terminate(nil) }
func (s *ReplaySubject[T]) terminate(err error) {
	s.mu.Lock()
	if s.terminal {
		s.mu.Unlock()
		return
	}
	s.terminal, s.err = true, err
	subs := snapshot(s.subs)
	s.subs = nil
	s.mu.Unlock()
	for _, sub := range subs {
		if err != nil {
			sub.Error(err)
		} else {
			sub.Complete()
		}
	}
}
func (s *ReplaySubject[T]) Observable() *Observable[T] {
	return create(func(ctx context.Context, dst *Observer[T]) func() {
		s.mu.Lock()
		values := append([]T(nil), s.values...)
		terminal, err := s.terminal, s.err
		var id uint64
		if !terminal {
			id = s.nextID
			s.nextID++
			s.subs[id] = dst
		}
		s.mu.Unlock()
		for _, v := range values {
			dst.Next(v)
		}
		if terminal {
			if err != nil {
				dst.Error(err)
			} else {
				dst.Complete()
			}
			return nil
		}
		var once sync.Once
		cancel := func() { once.Do(func() { s.mu.Lock(); delete(s.subs, id); s.mu.Unlock() }) }
		go func() { <-ctx.Done(); cancel() }()
		return cancel
	})
}

// Share multicasts one upstream subscription while at least one subscriber is
// present. A new connection is created after termination or ref-count zero.
func Share[T any]() Operator[T] {
	var mu sync.Mutex
	var subject *Subject[T]
	var upstream *Subscription
	connected := false
	refs := 0
	return func(source *Observable[T]) *Observable[T] {
		return create(func(ctx context.Context, dst *Observer[T]) func() {
			mu.Lock()
			if subject == nil {
				subject = NewSubject[T]()
			}
			current := subject
			refs++
			localCancel := current.observer.subscribe(ctx, dst)
			connect := !connected
			if connect {
				connected = true
			}
			mu.Unlock()
			if connect {
				sub := source.SubscribeObserver(context.Background(), current.Next,
					func(err error) { current.Error(err); mu.Lock(); connected = false; upstream = nil; mu.Unlock() },
					func() { current.Complete(); mu.Lock(); connected = false; upstream = nil; mu.Unlock() })
				mu.Lock()
				if connected {
					upstream = sub
				} else {
					sub.Cancel()
				}
				mu.Unlock()
			}
			var once sync.Once
			return func() {
				once.Do(func() {
					localCancel()
					mu.Lock()
					refs--
					if refs == 0 {
						if upstream != nil {
							upstream.Cancel()
						}
						upstream = nil
						connected = false
						subject = nil
					}
					mu.Unlock()
				})
			}
		})
	}
}

func NewBehaviorSubject[T any](initial T) *BehaviorSubject[T] {
	replay := NewReplaySubject[T](1)
	replay.Next(initial)
	return &BehaviorSubject[T]{value: initial, replay: replay}
}
func (s *BehaviorSubject[T]) Next(v T)        { s.mu.Lock(); s.value = v; s.mu.Unlock(); s.replay.Next(v) }
func (s *BehaviorSubject[T]) Error(err error) { s.replay.Error(err) }
func (s *BehaviorSubject[T]) Complete()       { s.replay.Complete() }
func (s *BehaviorSubject[T]) Value() T        { s.mu.RLock(); defer s.mu.RUnlock(); return s.value }
func (s *BehaviorSubject[T]) Observable() *Observable[T] {
	return s.replay.Observable()
}
