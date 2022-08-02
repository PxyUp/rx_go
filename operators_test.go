package rx_go_test

import (
	"github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFirst(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch := obs.Pipe(rx_go.First[int]()).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1}, res)
}

func TestLast(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch := obs.Pipe(rx_go.Last[int]()).Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{6}, res)
}

func TestFilter(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch := obs.Pipe(rx_go.Filter[int](func(value int) bool {
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
	ch := obs.Pipe(rx_go.Map[int](func(value int) int {
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
	ch := obs.Pipe(rx_go.Map[int](func(value int) int {
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
