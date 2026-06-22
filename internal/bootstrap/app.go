package bootstrap

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/huypham67/bookmark-common/pkg/logger"
	"github.com/huypham67/bookmark-worker/internal/worker"
	"github.com/rs/zerolog/log"
)

// App manages the worker lifecycle.
type App struct {
	container *Container
}

// NewApp initializes logging and all worker dependencies.
func NewApp() (*App, error) {
	if err := logger.NewClient(""); err != nil {
		return nil, err
	}

	container, err := NewContainer()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize container")
		return nil, err
	}

	log.Info().
		Str("queue", container.Config.QueueKey).
		Int("workers", container.Config.WorkerCount).
		Msg("application initialized")

	return &App{container: container}, nil
}

// Run dispatches jobs to a worker pool until SIGINT/SIGTERM is received.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := a.container.Config
	pool := worker.NewPool(
		a.container.Subscriber,
		a.container.Handler,
		cfg.QueueKey,
		cfg.WorkerCount,
		cfg.JobBufferSize,
		cfg.PollInterval,
	)
	return pool.Run(ctx)
}

// Close gracefully shuts down all resources.
func (a *App) Close() error {
	return a.container.Close()
}
