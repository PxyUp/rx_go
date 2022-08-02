package rx_go_test

import (
	"github.com/PxyUp/rx_go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	values := []int{1, 2, 3, 4, 5, 6}
	obs := rx_go.From(values...)
	ch := obs.Subscribe()
	var res []int
	for val := range ch {
		res = append(res, val)
	}
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, res)
}
