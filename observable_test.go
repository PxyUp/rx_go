package rx_go_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rx "github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/require"
)

func collect[T any](t *testing.T, source *rx.Observable[T]) ([]T, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var values []T
	var streamErr error
	var mu sync.Mutex
	sub := source.SubscribeObserver(ctx, func(v T) { mu.Lock(); values = append(values, v); mu.Unlock() }, func(err error) { streamErr = err }, nil)
	select {
	case <-sub.Done():
	case <-ctx.Done():
		t.Fatal("observable did not terminate")
	}
	mu.Lock()
	defer mu.Unlock()
	return values, streamErr
}

func TestColdObservableSupportsRepeatedSubscriptions(t *testing.T) {
	source := rx.From(1, 2, 3)
	a, err := collect(t, source)
	require.NoError(t, err)
	b, err := collect(t, source)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, a)
	require.Equal(t, a, b)
}

func TestCancellationIsSubscriptionLocal(t *testing.T) {
	source := rx.NewInterval(2*time.Millisecond, true)
	ctxA, cancelA := context.WithCancel(context.Background())
	var a, b atomic.Int64
	sa := source.SubscribeObserver(ctxA, func(time.Time) {
		if a.Add(1) == 2 {
			cancelA()
		}
	}, nil, nil)
	ctxB, cancelB := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelB()
	sb := source.SubscribeObserver(ctxB, func(time.Time) { b.Add(1) }, nil, nil)
	<-sa.Done()
	<-sb.Done()
	require.GreaterOrEqual(t, b.Load(), int64(3))
}

func TestErrorContract(t *testing.T) {
	want := errors.New("boom")
	values, err := collect(t, rx.Throw[int](want))
	require.Empty(t, values)
	require.ErrorIs(t, err, want)
}

func TestCreationAndCombination(t *testing.T) {
	values, err := collect(t, rx.Concat(rx.From(1, 2), rx.From(3, 4)))
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3, 4}, values)
	values, err = collect(t, rx.Merge(rx.From(1, 2), rx.From(3, 4)))
	require.NoError(t, err)
	require.ElementsMatch(t, []int{1, 2, 3, 4}, values)
	slices, err := collect(t, rx.ToSlice(rx.From(1, 2, 3)))
	require.NoError(t, err)
	require.Equal(t, [][]int{{1, 2, 3}}, slices)
	pairs, err := collect(t, rx.Pairwise(rx.From(1, 2, 3)))
	require.NoError(t, err)
	require.Equal(t, [][2]int{{1, 2}, {2, 3}}, pairs)
}

func TestReduceScanAndSwitchSemantics(t *testing.T) {
	reduced, err := collect(t, rx.Reduce(rx.From(1, 2, 3), func(a, b int) int { return a + b }, 0))
	require.NoError(t, err)
	require.Equal(t, []int{6}, reduced)
	scanned, err := collect(t, rx.From(1, 2, 3).Pipe(rx.Scan(func(a, b int) int { return a + b }, 0)))
	require.NoError(t, err)
	require.Equal(t, []int{1, 3, 6}, scanned)
	switched, err := collect(t, rx.Switch(rx.From(1, 2, 3), func(v int) *rx.Observable[string] {
		return rx.MapTo(rx.Timer(10*time.Millisecond), func(time.Time) string { return fmt.Sprint(v) })
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"3"}, switched)
}

func TestForkJoin(t *testing.T) {
	values, err := collect(t, rx.ForkJoin(rx.From(1, 2), rx.From(3, 4)))
	require.NoError(t, err)
	require.Equal(t, [][]int{{2, 4}}, values)
	empty, err := collect(t, rx.ForkJoin(rx.From(1), rx.EmptyOf[int]()))
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestHTTPIsLazyAndPropagatesErrors(t *testing.T) {
	var calls atomic.Int64
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
	})}
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	source, err := rx.NewHttp(client, req)
	require.NoError(t, err)
	require.Zero(t, calls.Load())
	values, streamErr := collect(t, source)
	require.NoError(t, streamErr)
	require.Equal(t, [][]byte{[]byte("ok")}, values)
	require.Equal(t, int64(1), calls.Load())
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
