package rate_limiter

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

type CircuitBreaker interface {
	Open() bool // returns true if the circuit is open, and will block until the circuit is closed
	// HalfOpen() bool // todo: implement
	Closed() bool // returns true if the circuit is closed, and which is the default state
}

// Window defines a sliding window for use to count averages over time.
type Window struct {
	mu             sync.RWMutex
	windowDuration time.Duration
	bucketDuration time.Duration
	numBuckets     int
	buckets        []int
	pos            int
	size           int
	threshold      int
	stopping       chan struct{}
}

func newWindow(windowDuration time.Duration, numBuckets int, threshold int) (*Window, error) {
	if windowDuration == 0 {
		return nil, errors.New("windowDuration cannot be zero")
	}
	if numBuckets == 0 {
		return nil, errors.New("subWindow cannot be zero")
	}
	if int(windowDuration) <= numBuckets || int(windowDuration)%numBuckets != 0 {
		return nil, errors.New("windowDuration has to be a multiplier of numBuckets")
	}

	sw := &Window{
		windowDuration: windowDuration,
		bucketDuration: windowDuration / time.Duration(numBuckets),
		numBuckets:     numBuckets,
		buckets:        make([]int, numBuckets),
		stopping:       make(chan struct{}, 1),
		size:           1,
		threshold:      threshold,
	}

	return sw, nil
}

func StartWindow(windowDuration time.Duration, numBuckets int, threshold int) (*Window, error) {
	sw, err := newWindow(windowDuration, numBuckets, threshold)
	if err != nil {
		return nil, err
	}
	go sw.Start()
	return sw, nil
}

func StartWindowFromSampleBuckets(windowDuration time.Duration, numBuckets int, threshold int, samples []int) (*Window, error) {
	sw, err := newWindow(windowDuration, numBuckets, threshold)
	if err != nil {
		return nil, err
	}

	sw.fillBuckets(samples...)
	go sw.Start()
	return sw, nil
}

func (sw *Window) Start() {
	ticker := time.NewTicker(sw.bucketDuration)

	for {
		select {
		case <-ticker.C:
			sw.nextPosition()
		case <-sw.stopping:
			return
		}
	}
}

func (sw *Window) nextPosition() {
	sw.mu.Lock()

	sw.pos = (sw.pos + 1) % sw.numBuckets
	sw.buckets[sw.pos] = 0

	if sw.size < sw.numBuckets {
		sw.size++
	}

	sw.mu.Unlock()
}

func (sw *Window) Add(v int) {
	//log.Printf("Add %v\n", v)
	sw.mu.Lock()
	sw.buckets[sw.pos] += v
	sw.mu.Unlock()
}

func (sw *Window) Increment() {
	//log.Println("increment")
	sw.mu.Lock()
	sw.buckets[sw.pos] += 1
	sw.mu.Unlock()
}

// Last retrieves the last N buckets and returns the total counts and number of buckets
func (sw *Window) Last(n int) (total int) {
	if n <= 0 {
		return 0
	}

	sw.mu.RLock()
	defer sw.mu.RUnlock()

	if n > sw.size {
		n = sw.size
	}

	var result int

	for i := 0; i < n; i++ {
		result += sw.buckets[(sw.numBuckets+sw.pos-i)%sw.numBuckets]
	}

	return result
}

func (sw *Window) Count() int {
	return sw.Last(sw.numBuckets)
}

func (sw *Window) Open() bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.Count() >= sw.threshold
}

func (sw *Window) Closed() bool {
	return !sw.Open()
}

func (sw *Window) fillBuckets(i ...int) {
	for j, v := range i {
		sw.Add(v)
		if j != len(i)-1 {
			sw.nextPosition()
		}
	}
}
