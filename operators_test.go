package rx_go_test

import (
	"context"
	"github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	values := []int{1, 2, 3}
	ch, _ := rx_go.From(values...).Pipe(rx_go.Repeat[int](2)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 1, 2, 2, 3, 3}, res)
}

func TestSleep(t *testing.T) {
	values := []int{1, 2, 3}
	ch, _ := rx_go.From(values...).Pipe(rx_go.Delay[int](time.Second), rx_go.Debounce[int](time.Millisecond*500)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3}, res)
}

func TestDebounce(t *testing.T) {
	values := []int{1, 2, 3}
	ch, _ := rx_go.From(values...).Pipe(rx_go.Debounce[int](time.Second)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{3}, res)
}

func TestEndWith(t *testing.T) {
	ch, _ := rx_go.From([]int{1}...).Pipe(rx_go.EndWith(2)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2}, res)
}

func TestStartWith(t *testing.T) {
	ch, _ := rx_go.From([]int{1}...).Pipe(rx_go.StartWith(2)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{2, 1}, res)
}

func TestSkipUntil_Never(t *testing.T) {
	ch, cancel := rx_go.From([]int{1, 2, 3}...).Pipe(
		rx_go.SkipUntil[int, any](rx_go.Never),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Nil(t, res)
	cancel()
}

func TestSkipUntilCtx_Never(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	ch, cancel := rx_go.From([]int{1, 2, 3}...).Pipe(
		rx_go.SkipUntilCtx[int](ctx),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Nil(t, res)
	cancel()
}

func TestSkipUntilCtx_Emitted(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	ch, cancel := rx_go.From([]int{1, 2, 3}...).Pipe(
		rx_go.Do(func(value int) {
			if value == 2 {
				cancelCtx()
			}
		}),
		rx_go.SkipUntilCtx[int](ctx),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{2, 3}, res)
	cancel()
}

func TestSkipUntil_Emitted(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	ch, cancel := rx_go.From([]int{1, 2, 3}...).Pipe(
		rx_go.AfterCtx[int](ctx),
		rx_go.SkipUntil[int, int](rx_go.Of(1).Pipe(rx_go.Do(func(value int) {
			cancelCtx()
		}))),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3}, res)
	cancel()
}

func TestSkip(t *testing.T) {
	ch, _ := rx_go.From([]int{1, 2, 3}...).Pipe(rx_go.Skip[int](2)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{3}, res)
}

func TestAfterCtx(t *testing.T) {
	values := []int{1, 2, 3}
	ctx, cancel := context.WithCancel(context.Background())
	ch, _ := rx_go.From(values...).Pipe(
		rx_go.AfterCtx[int](ctx),
	).Subscribe()
	cancel()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3}, res)
}

func TestUntilCtx(t *testing.T) {
	values := []int{1, 2, 3}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, _ := rx_go.From(values...).Pipe(
		rx_go.Do(func(value int) {
			if value == 2 {
				cancel()
			}
		}),
		rx_go.UntilCtx[int](ctx),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2}, res)
}

func TestDo(t *testing.T) {
	values := []int{1, 2, 3}
	count := 0
	ch, _ := rx_go.From(values...).Pipe(
		rx_go.Do(func(_ int) {
			count += 1
		}),
	).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3}, res)
	assert.Equal(t, 3, count)
}

func TestDistinctWith(t *testing.T) {
	values := []int{1, 2, 2, 2, 3, 4, 4, 4, 4}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.DistinctWith[int](func(a, b int) bool { return a == b })).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3, 4}, res)
}

func TestDistinct(t *testing.T) {
	values := []int{1, 2, 2, 2, 3, 4, 4, 4, 4}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.Distinct[int]()).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3, 4}, res)
}

func TestTake(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.Take[int](3)).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3}, res)
}

func TestFirstOne(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.FirstOne[int]()).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1}, res)
}

func TestLastOne(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.LastOne[int]()).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{6}, res)
}

func TestFilter(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.Filter[int](func(value int) bool {
		return value > 3
	})).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{4, 5, 6}, res)
}

func TestMap(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.Map[int](func(value int) int {
		return value * 3
	})).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{3, 6, 9, 12, 15, 18}, res)
}

func TestFilterMap(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Pipe(rx_go.Map[int](func(value int) int {
		return value * 3
	}), rx_go.Filter[int](func(value int) bool {
		return value > 16
	})).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{18}, res)
}
