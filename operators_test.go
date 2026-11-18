package rx_go_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	rx "github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/require"
)

func TestBasicOperators(t *testing.T) {
	source := rx.From(1, 2, 2, 3, 4).Pipe(rx.Map(func(v int) int { return v * 2 }), rx.Filter(func(v int) bool { return v >= 4 }), rx.Distinct[int](), rx.Skip[int](1), rx.Take[int](2))
	values, err := collect(t, source)
	require.NoError(t, err)
	require.Equal(t, []int{6, 8}, values)
}

func TestDistinctVariants(t *testing.T) {
	all, _ := collect(t, rx.From(1, 2, 1, 2).Pipe(rx.Distinct[int]()))
	require.Equal(t, []int{1, 2}, all)
	adjacent, _ := collect(t, rx.From(1, 2, 1, 1, 2).Pipe(rx.DistinctUntilChanged[int]()))
	require.Equal(t, []int{1, 2, 1, 2}, adjacent)
}

func TestRepeatRepeatsSequence(t *testing.T) {
	values, err := collect(t, rx.From(1, 2, 3).Pipe(rx.Repeat[int](2)))
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3, 1, 2, 3}, values)
}

func TestFirstLastFindAndElementAt(t *testing.T) {
	first, _ := collect(t, rx.From(1, 2, 3).Pipe(rx.FirstOne[int]()))
	require.Equal(t, []int{1}, first)
	last, _ := collect(t, rx.From(1, 2, 3).Pipe(rx.LastOne[int]()))
	require.Equal(t, []int{3}, last)
	found, _ := collect(t, rx.From(1, 2, 3).Pipe(rx.Find(func(v int) bool { return v > 1 })))
	require.Equal(t, []int{2}, found)
	element, _ := collect(t, rx.From(1, 2, 3).Pipe(rx.ElementAt[int](1)))
	require.Equal(t, []int{2}, element)
	_, err := collect(t, rx.From(1).Pipe(rx.ElementAt[int](3)))
	require.Error(t, err)
}

func TestDelayDebounceAndTimeout(t *testing.T) {
	start := time.Now()
	values, err := collect(t, rx.Of(1).Pipe(rx.Delay[int](15*time.Millisecond)))
	require.NoError(t, err)
	require.Equal(t, []int{1}, values)
	require.GreaterOrEqual(t, time.Since(start), 15*time.Millisecond)
	values, err = collect(t, rx.From(1, 2, 3).Pipe(rx.Debounce[int](5*time.Millisecond)))
	require.NoError(t, err)
	require.Equal(t, []int{3}, values)
	_, err = collect(t, rx.NeverOf[int]().Pipe(rx.Timeout[int](5*time.Millisecond)))
	require.ErrorIs(t, err, rx.ErrTimeout)
}

func TestCatchRetryAndFinally(t *testing.T) {
	want := errors.New("failure")
	recovered, err := collect(t, rx.Throw[int](want).Pipe(rx.Catch(func(error) *rx.Observable[int] { return rx.Of(7) })))
	require.NoError(t, err)
	require.Equal(t, []int{7}, recovered)
	var attempts atomic.Int64
	source := rx.Defer(func() *rx.Observable[int] {
		if attempts.Add(1) < 3 {
			return rx.Throw[int](want)
		}
		return rx.Of(9)
	}).Pipe(rx.Retry[int](2))
	values, err := collect(t, source)
	require.NoError(t, err)
	require.Equal(t, []int{9}, values)
	require.Equal(t, int64(3), attempts.Load())
	var finalized atomic.Int64
	_, _ = collect(t, rx.Of(1).Pipe(rx.Finally[int](func() { finalized.Add(1) })))
	require.Equal(t, int64(1), finalized.Load())
}

func TestContextOperators(t *testing.T) {
	gate, open := context.WithCancel(context.Background())
	open()
	values, err := collect(t, rx.From(1, 2).Pipe(rx.AfterCtx[int](gate)))
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, values)
	values, err = collect(t, rx.From(1, 2).Pipe(rx.SkipUntilCtx[int](gate)))
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, values)
}
