package bootstrap

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/huypham67/bookmark-common/pkg/logger"
	pkgRedis "github.com/huypham67/bookmark-common/pkg/redis"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
	"github.com/huypham67/bookmark-worker/internal/worker"
)

// App is the worker. It dequeues jobs from a Redis list and dispatches them to
// the import handler. No pool, no graceful drain — a single poll loop, kept
// deliberately simple.
type App struct {
	cfg        *Config
	redis      *redis.Client
	subscriber queue.Subscriber
	handler    bookmarkHandler.Handler
}

// NewApp initializes logging, config, the Redis client and the job pipeline.
func NewApp() (*App, error) {
	if err := logger.NewClient(""); err != nil {
		return nil, err
	}

	cfg, err := NewConfig()
	if err != nil {
		log.Error().Err(err).Msg("failed to load config")
		return nil, err
	}

	rdb, err := pkgRedis.NewClient("")
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize redis client")
		return nil, err
	}

	log.Info().Str("queue", cfg.QueueKey).Msg("application initialized successfully")

	return &App{
		cfg:        cfg,
		redis:      rdb,
		subscriber: initRedisSubscriber(rdb),
		handler:    initImportHandler(),
	}, nil
}

func initImportHandler() bookmarkHandler.Handler {
	repo := bookmarkRepo.NewRepository()
	svc := bookmarkSvc.NewService(repo)
	return bookmarkHandler.NewHandler(svc)
}

func initRedisSubscriber(rdb *redis.Client) queue.Subscriber {
	return queue.NewRedisSubscriber(rdb)
}

// Run dispatches jobs to a worker pool until SIGINT/SIGTERM is received, then
// drains in-flight jobs before returning.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool := worker.NewPool(a.subscriber, a.handler, a.cfg.QueueKey, a.cfg.WorkerCount, a.cfg.JobBufferSize)

	log.Info().Int("workers", a.cfg.WorkerCount).Msg("worker started, polling for jobs...")
	return pool.Run(ctx)
}
