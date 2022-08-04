package rx_go_test

import (
	"fmt"
	"github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSwitch(t *testing.T) {
	ch, _ := rx_go.Switch(rx_go.From([]int{1, 2, 3}...), func(value int) *rx_go.Observable[string] {
		return rx_go.From(fmt.Sprintf("HELLO %d", value)).Pipe(rx_go.Repeat[string](2))
	}).Subscribe()
	var res []string
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []string{"HELLO 1", "HELLO 1", "HELLO 2", "HELLO 2", "HELLO 3", "HELLO 3"}, res)
}

func TestMapTo(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	ch, _ := rx_go.MapTo[int, string](rx_go.From(values...), func(t int) string {
		return fmt.Sprintf("hello %d", t)
	}).Subscribe()
	var res []string
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []string{"hello 1", "hello 2", "hello 3", "hello 4", "hello 5", "hello 6"}, res)
}

func TestOf(t *testing.T) {
	ch, _ := rx_go.Of("hello").Subscribe()
	assert.Equal(t, "hello", <-ch)
}

func TestConcat(t *testing.T) {
	ch, _ := rx_go.Concat(rx_go.From([]int{1, 2, 3, 4, 5, 6}...)).Subscribe()
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, <-ch)
}

func TestReduce(t *testing.T) {
	ch, _ := rx_go.Reduce(rx_go.From([]int{1, 2, 3, 4, 5, 6}...), func(y string, t int) string {
		return y + fmt.Sprintf("%d", t)
	}, "").Subscribe()
	var res []string
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []string{
		"1",
		"12",
		"123",
		"1234",
		"12345",
		"123456",
	}, res)
}

func TestNew(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch, _ := obs.Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, res)
}

func TestObservable_Subscribe(t *testing.T) {
	obs := rx_go.From([]int{1, 2, 3, 4, 5, 6}...)
	ch, cancel := obs.Pipe(rx_go.Delay[int](time.Millisecond * 500)).Subscribe()
	go func() {
		time.Sleep(time.Millisecond * 700)
		cancel()
	}()
	var res []int
	for v := range ch {
		res = append(res, v)
	}
	assert.Equal(t, []int{1}, res)
}

func TestNewInterval(t *testing.T) {
	ch, cancel := rx_go.NewInterval(time.Millisecond*300, true).Subscribe()
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	var res []time.Time
	for v := range ch {
		res = append(res, v)
	}
	assert.Len(t, res, 4)
}

func TestMerge(t *testing.T) {
	ch, _ := rx_go.Merge[int](rx_go.From[int]([]int{1, 2, 3, 7}...), rx_go.From[int]([]int{4, 5, 6}...)).Subscribe()
	var res []int
	for v := range ch {
		res = append(res, v)
	}
	assert.Contains(t, res, 1)
	assert.Contains(t, res, 2)
	assert.Contains(t, res, 3)
	assert.Contains(t, res, 4)
	assert.Contains(t, res, 5)
	assert.Contains(t, res, 6)
	assert.Contains(t, res, 7)
	assert.Len(t, res, 7)
}

func TestFromChannel(t *testing.T) {
	intChan := make(chan int)
	go func() {
		intChan <- 1
		intChan <- 2
		close(intChan)
	}()

	ch, _ := rx_go.FromChannel(intChan).Subscribe()
	var res []int
	for v := range ch {
		res = append(res, v)
	}
	assert.Contains(t, res, 1)
	assert.Contains(t, res, 2)
	assert.Len(t, res, 2)
}
