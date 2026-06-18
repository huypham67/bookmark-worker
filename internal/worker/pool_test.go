package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
	queueMocks "github.com/huypham67/bookmark-worker/internal/repository/queue/mocks"
	"github.com/stretchr/testify/assert"
)

// handlerFn is a function-based Handler test double.
// Used instead of the generated mock because fine-grained panic control
// (e.g. panic on first call, succeed on second) is awkward with testify mock.
type handlerFn func(context.Context, []byte) error

func (h handlerFn) Handle(ctx context.Context, payload []byte) error { return h(ctx, payload) }

var _ bookmarkHandler.Handler = (handlerFn)(nil)

func newTestPool(t *testing.T, sub queue.Subscriber, h bookmarkHandler.Handler, workers, bufSize int) *Pool {
	t.Helper()
	return NewPool(sub, h, "test-queue", workers, bufSize, time.Millisecond)
}

func TestPool_Run(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		workers  int
		bufSize  int
		preStart func(*Pool)
		build    func(*testing.T, context.Context) (queue.Subscriber, bookmarkHandler.Handler, <-chan struct{}, func(*testing.T))
	}{
		{
			name:    "processes all dequeued jobs",
			workers: 2,
			bufSize: 10,
			build: func(t *testing.T, ctx context.Context) (queue.Subscriber, bookmarkHandler.Handler, <-chan struct{}, func(*testing.T)) {
				var handled atomic.Int32
				done := make(chan struct{})

				h := handlerFn(func(_ context.Context, _ []byte) error {
					if handled.Add(1) == 3 {
						close(done)
					}
					return nil
				})

				sub := queueMocks.NewSubscriber(t)
				sub.On("Dequeue", ctx, "test-queue").Return([]byte("j1"), nil).Once()
				sub.On("Dequeue", ctx, "test-queue").Return([]byte("j2"), nil).Once()
				sub.On("Dequeue", ctx, "test-queue").Return([]byte("j3"), nil).Once()
				sub.On("Dequeue", ctx, "test-queue").Return(nil, queue.ErrQueueEmpty)

				verify := func(t *testing.T) {
					assert.Equal(t, int32(3), handled.Load())
				}

				return sub, h, done, verify
			},
		},
		{
			name:    "graceful shutdown drains jobs already in the buffer",
			workers: 1,
			bufSize: 3,
			preStart: func(p *Pool) {
				p.jobs <- []byte("a")
				p.jobs <- []byte("b")
				p.jobs <- []byte("c")
			},
			build: func(t *testing.T, ctx context.Context) (queue.Subscriber, bookmarkHandler.Handler, <-chan struct{}, func(*testing.T)) {
				var handled atomic.Int32

				h := handlerFn(func(_ context.Context, _ []byte) error {
					handled.Add(1)
					return nil
				})

				sub := queueMocks.NewSubscriber(t)
				sub.On("Dequeue", ctx, "test-queue").Return(nil, queue.ErrQueueEmpty)

				// Cancel immediately — pool must still drain the 3 pre-filled jobs.
				done := make(chan struct{})
				close(done)

				verify := func(t *testing.T) {
					assert.Equal(t, int32(3), handled.Load())
				}

				return sub, h, done, verify
			},
		},
		{
			name:    "worker recovers from panic and processes next job",
			workers: 1,
			bufSize: 10,
			build: func(t *testing.T, ctx context.Context) (queue.Subscriber, bookmarkHandler.Handler, <-chan struct{}, func(*testing.T)) {
				var handled atomic.Int32
				var panicked atomic.Bool
				done := make(chan struct{})

				h := handlerFn(func(_ context.Context, payload []byte) error {
					if string(payload) == "bad" && panicked.CompareAndSwap(false, true) {
						panic(errors.New("simulated panic"))
					}
					if handled.Add(1) == 1 {
						close(done)
					}
					return nil
				})

				sub := queueMocks.NewSubscriber(t)
				sub.On("Dequeue", ctx, "test-queue").Return([]byte("bad"), nil).Once()
				sub.On("Dequeue", ctx, "test-queue").Return([]byte("good"), nil).Once()
				sub.On("Dequeue", ctx, "test-queue").Return(nil, queue.ErrQueueEmpty)

				verify := func(t *testing.T) {
					assert.True(t, panicked.Load(), "worker should have panicked")
					assert.Equal(t, int32(1), handled.Load(), "good job processed after recovery")
				}

				return sub, h, done, verify
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			sub, h, done, verify := tc.build(t, ctx)
			pool := newTestPool(t, sub, h, tc.workers, tc.bufSize)

			if tc.preStart != nil {
				tc.preStart(pool)
			}

			go func() {
				<-done
				cancel()
			}()

			pool.Run(ctx)
			verify(t)
		})
	}
}
