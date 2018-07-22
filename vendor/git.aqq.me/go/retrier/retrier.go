// Package retrier for retry any operations by timer.
package retrier

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

func init() {
	now := time.Now()
	rand.Seed(now.Unix())
}

// Status type is used for all returned statuses
type Status int

// Retrier type represent a single retrier.
type Retrier struct {
	config   Config
	idSeq    int
	attempts map[int]int
	stop     chan struct{}
	stopped  bool
	m        *sync.Mutex
}

// Config reperesents configuration parameters of retrier.
type Config struct {
	MaxAttempts int
	RetryPolicy []time.Duration
}

const (
	// Succeed status must be returned on success operation.
	Succeed Status = iota

	// NeedRetry status must be returned when operation needs to be repeated.
	NeedRetry

	// Failed status must be returned on failed operation.
	Failed
)

var defaultRetryPolicy = []time.Duration{time.Second * 5}

var (
	// ErrFailed returned when operation was failed.
	ErrFailed = errors.New("operation failed and can not be retried")

	// ErrMaxAttempts returned when maximum number of attempts of operation will
	// be reached.
	ErrMaxAttempts = errors.New("maximum number of attempts reached")

	// ErrStopped returned if performing operation on stopped retrier.
	ErrStopped = errors.New("retries stopped")
)

// New function creates new retrier.
func New(config Config) *Retrier {
	if config.RetryPolicy == nil {
		config.RetryPolicy = defaultRetryPolicy
	}

	return &Retrier{
		config:   config,
		idSeq:    1,
		attempts: make(map[int]int),
		stop:     make(chan struct{}),
		m:        &sync.Mutex{},
	}
}

// Do function executes an operation and retry it if operation was failed.
func (r *Retrier) Do(f func() Status) error {
	r.m.Lock()

	if r.stopped {
		r.m.Unlock()
		return ErrStopped
	}

	id := r.idSeq
	r.idSeq++

	r.m.Unlock()

	for {
		status := f()

		r.m.Lock()
		r.attempts[id]++
		attempts := r.attempts[id]
		r.m.Unlock()

		if status == Succeed {
			r.m.Lock()
			delete(r.attempts, id)
			r.m.Unlock()

			return nil
		} else if status == NeedRetry {
			if r.config.MaxAttempts == 0 ||
				attempts < r.config.MaxAttempts {

				interval := r.getNextInterval(attempts)
				t := time.NewTimer(interval)

				select {
				case <-r.stop:
					t.Stop()
					return ErrStopped
				case <-t.C:
				}
			} else {
				r.m.Lock()
				delete(r.attempts, id)
				r.m.Unlock()

				return ErrMaxAttempts
			}
		} else {
			r.m.Lock()
			delete(r.attempts, id)
			r.m.Unlock()

			return ErrFailed
		}
	}
}

// Stop method stops all retries.
func (r *Retrier) Stop() {
	r.m.Lock()
	defer r.m.Unlock()

	if r.stopped {
		return
	}

	close(r.stop)
	r.stopped = true
}

func (r *Retrier) getNextInterval(attemptNum int) time.Duration {
	i := attemptNum - 1
	maxIndex := len(r.config.RetryPolicy) - 1

	if i > maxIndex {
		i = maxIndex
	}

	interval := int64(r.config.RetryPolicy[i])

	return time.Duration(interval/2 + rand.Int63n(interval))
}
