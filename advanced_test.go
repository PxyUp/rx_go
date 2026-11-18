package rx_go_test

import (
	"context"
	"sync"
	"testing"
	"time"

	rx "github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/require"
)

func TestFlatMapConcatMapAndCombineLatest(t *testing.T) {
	flat, err := collect(t, rx.FlatMap(rx.From(1, 2), func(v int) *rx.Observable[int] { return rx.From(v, v*10) }))
	require.NoError(t, err)
	require.ElementsMatch(t, []int{1, 10, 2, 20}, flat)
	concat, err := collect(t, rx.ConcatMap(rx.From(1, 2), func(v int) *rx.Observable[int] { return rx.From(v, v*10) }))
	require.NoError(t, err)
	require.Equal(t, []int{1, 10, 2, 20}, concat)
	latest, err := collect(t, rx.CombineLatest(rx.Of(2), rx.Of(3), func(a, b int) int { return a + b }))
	require.NoError(t, err)
	require.Equal(t, []int{5}, latest)
	zipped, err := collect(t, rx.Zip(rx.From(1, 2), rx.From("a", "b")))
	require.NoError(t, err)
	require.Equal(t, []rx.Pair[int, string]{{First: 1, Second: "a"}, {First: 2, Second: "b"}}, zipped)
	counts, err := collect(t, rx.Count(rx.From(1, 2, 3)))
	require.NoError(t, err)
	require.Equal(t, []int{3}, counts)
}

func TestSubjects(t *testing.T) {
	subject := rx.NewSubject[int]()
	var values []int
	var mu sync.Mutex
	sub := subject.Observable().SubscribeObserver(context.Background(), func(v int) { mu.Lock(); values = append(values, v); mu.Unlock() }, nil, nil)
	subject.Next(1)
	subject.Next(2)
	subject.Complete()
	<-sub.Done()
	require.Equal(t, []int{1, 2}, values)

	behavior := rx.NewBehaviorSubject(5)
	got, err := collect(t, behavior.Observable().Pipe(rx.Take[int](1)))
	require.NoError(t, err)
	require.Equal(t, []int{5}, got)
	replay := rx.NewReplaySubject[int](2)
	replay.Next(1)
	replay.Next(2)
	replay.Next(3)
	replay.Complete()
	got, err = collect(t, replay.Observable())
	require.NoError(t, err)
	require.Equal(t, []int{2, 3}, got)
}

func TestShareUsesSingleUpstream(t *testing.T) {
	var mu sync.Mutex
	subscriptions := 0
	source := rx.Create(func(ctx context.Context, dst *rx.Observer[int]) func() {
		mu.Lock()
		subscriptions++
		mu.Unlock()
		go func() { dst.Next(1); time.Sleep(10 * time.Millisecond); dst.Complete() }()
		return nil
	}).Pipe(rx.Share[int]())
	a := source.SubscribeObserver(context.Background(), func(int) {}, nil, nil)
	b := source.SubscribeObserver(context.Background(), func(int) {}, nil, nil)
	<-a.Done()
	<-b.Done()
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, subscriptions)
}
