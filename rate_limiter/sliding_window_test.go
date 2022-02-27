package rate_limiter

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestWindow_Simple(t *testing.T) {
	win, err := StartWindowFromSampleBuckets(time.Second*5, 5, 10, []int{1, 2, 3})
	assert.NilError(t, err)

	assert.Equal(t, win.Last(1), 3)
	assert.Equal(t, win.Closed(), true)
	assert.Equal(t, win.Open(), false)
}

func TestWindow_LoopOver(t *testing.T) {
	win, err := StartWindowFromSampleBuckets(time.Second*5, 5, 10, []int{1, 2, 3, 4, 5})
	assert.NilError(t, err)
	win.pos = 3

	assert.Equal(t, win.Last(5), 15)
	assert.Equal(t, win.Closed(), false)
	assert.Equal(t, win.Open(), true)
}

func TestWindow_Add(t *testing.T) {
	win, err := StartWindow(time.Second*5, 5, 100)
	assert.NilError(t, err)
	win.Add(123)
	win.Add(456)

	assert.Equal(t, win.Last(1), 123+456)
	assert.Equal(t, win.Count(), 123+456)
	assert.Equal(t, win.Closed(), false)
	assert.Equal(t, win.Open(), true)
}

func TestWindow_NextPosition(t *testing.T) {
	win, err := StartWindow(time.Second*5, 5, 100)
	assert.NilError(t, err)

	win.Add(123)
	win.nextPosition()
	win.Add(456)
	win.nextPosition()
	win.Add(789)

	assert.Equal(t, win.Last(1), 789)
	assert.Equal(t, win.Count(), 123+456+789)
	assert.Equal(t, win.Closed(), false)
	assert.Equal(t, win.Open(), true)
}
