package worker

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
)

// Pool dequeues jobs with a single poller goroutine and dispatches them to a
// fixed set of worker goroutines over a buffered channel. The buffer provides
// backpressure: the poller stops pulling from Redis once the buffer is full.
type Pool struct {
	subscriber   queue.Subscriber
	handler      bookmarkHandler.Handler
	queueKey     string
	workerCount  int
	pollInterval time.Duration
	jobs         chan []byte
	wg           sync.WaitGroup
}

// NewPool wires a pool with workerCount workers and a buffered job channel.
func NewPool(sub queue.Subscriber, h bookmarkHandler.Handler, queueKey string, workerCount, bufferSize int) *Pool {
	return &Pool{
		subscriber:   sub,
		handler:      h,
		queueKey:     queueKey,
		workerCount:  workerCount,
		pollInterval: time.Second,
		jobs:         make(chan []byte, bufferSize),
	}
}

// Run starts the workers and blocks polling until ctx is cancelled, then drains
// in-flight jobs before returning.
func (p *Pool) Run(ctx context.Context) error {
	for i := 0; i < p.workerCount; i++ {
		w := NewWorker(i+1, p.jobs, p.handler) // worker IDs are 1-indexed for logging
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.supervise(w)
		}()
	}

	p.poll(ctx) // blocks until ctx.Done()

	close(p.jobs) // no more jobs -> workers exit their range loop
	p.wg.Wait()   // wait for buffered jobs to drain
	return nil
}

// poll pulls jobs off the queue and feeds them to the workers. It sleeps only
// when the queue is empty or on error; otherwise it dispatches as fast as the
// workers can consume (bounded by the channel buffer).
func (p *Pool) poll(ctx context.Context) {
	for {
		payload, err := p.subscriber.Dequeue(ctx, p.queueKey)
		switch {
		case err == nil:
			select {
			case p.jobs <- payload: // blocks if buffer full = backpressure
			case <-ctx.Done():
				return
			}
		case errors.Is(err, queue.ErrQueueEmpty), errors.Is(err, context.Canceled):
			if !p.sleep(ctx) {
				return
			}
		default:
			log.Error().Err(err).Msg("failed to dequeue job")
			if !p.sleep(ctx) {
				return
			}
		}
	}
}

// supervise runs a worker and restarts it whenever it dies from a panic, so a
// single bad job cannot permanently shrink the pool. It returns only when the
// worker exits normally (the job channel is closed during shutdown).
func (p *Pool) supervise(w *Worker) {
	for {
		if p.runWorker(w) {
			return
		}
		log.Warn().Int("worker_id", w.id).Msg("worker restarting after panic")
	}
}

// runWorker runs w.Run guarded by recover. It returns true when the worker
// exits normally (channel closed) and false when a panic stopped it.
func (p *Pool) runWorker(w *Worker) (exitedNormally bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Int("worker_id", w.id).
				Msg("worker panicked")
			exitedNormally = false
		}
	}()

	w.Run()
	return true
}

// sleep waits for the poll interval, returning false if ctx is cancelled first.
func (p *Pool) sleep(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(p.pollInterval):
		return true
	}
}
