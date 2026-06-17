package worker

import (
	"context"

	"github.com/rs/zerolog/log"

	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
)

// Worker consumes jobs from a receive-only channel and dispatches each to the
// handler. It owns no lifecycle: it runs until the channel is closed, leaving
// goroutine management and shutdown to the Pool.
type Worker struct {
	id      int
	jobs    <-chan []byte
	handler bookmarkHandler.Handler
}

// NewWorker creates a worker bound to the shared job channel.
func NewWorker(id int, jobs <-chan []byte, h bookmarkHandler.Handler) *Worker {
	return &Worker{
		id:      id,
		jobs:    jobs,
		handler: h,
	}
}

// Run consumes jobs until the channel is closed.
func (w *Worker) Run() {
	for payload := range w.jobs {
		log.Info().Int("worker_id", w.id).Str("payload", string(payload)).Msg("worker picked up job")

		// Detached context so a job already popped from Redis finishes during
		// the shutdown drain instead of being canceled and lost.
		if err := w.handler.Handle(context.Background(), payload); err != nil {
			log.Error().Err(err).Int("worker_id", w.id).Msg("failed to handle job")
		}
	}
}
