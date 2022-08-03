package rx_go_test

import (
	"fmt"
	"github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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
