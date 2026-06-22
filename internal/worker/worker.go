package worker

import (
	"context"

	"github.com/rs/zerolog/log"
)

// Handler processes a raw queue payload.
type Handler interface {
	Handle(ctx context.Context, payload []byte) error
}

// Worker consumes jobs from a receive-only channel and dispatches each to the
// handler. It owns no lifecycle: it runs until the channel is closed, leaving
// goroutine management and shutdown to the Pool.
type Worker struct {
	id      int
	jobs    <-chan []byte
	handler Handler
}

// NewWorker creates a Worker that reads from the shared jobs channel.
func NewWorker(id int, jobs <-chan []byte, h Handler) *Worker {
	return &Worker{
		id:      id,
		jobs:    jobs,
		handler: h,
	}
}

// Run processes jobs from the channel until it is closed.
func (w *Worker) Run() {
	for payload := range w.jobs {
		log.Info().Int("worker_id", w.id).Msg("received job")
		if err := w.handler.Handle(context.Background(), payload); err != nil {
			log.Error().Err(err).Int("worker_id", w.id).Msg("failed to handle job")
		}
	}
}
