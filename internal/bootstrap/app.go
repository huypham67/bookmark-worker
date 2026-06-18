package bootstrap

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/huypham67/bookmark-common/pkg/logger"
	pkgRedis "github.com/huypham67/bookmark-common/pkg/redis"
	"github.com/huypham67/bookmark-common/pkg/sqldb"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
	cacheRepo "github.com/huypham67/bookmark-worker/internal/repository/cache"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
	"github.com/huypham67/bookmark-worker/internal/worker"
)

// App wires all worker dependencies and runs the job processing loop.
type App struct {
	cfg        *Config
	redis      *redis.Client
	subscriber queue.Subscriber
	handler    bookmarkHandler.Handler
}

// NewApp initializes logging, config, Redis and DB clients, and the full job processing pipeline.
func NewApp() (*App, error) {
	if err := logger.NewClient(""); err != nil {
		return nil, err
	}

	cfg, err := NewConfig()
	if err != nil {
		return nil, err
	}

	rdb, err := pkgRedis.NewClient("")
	if err != nil {
		return nil, err
	}

	db, err := sqldb.NewClient("")
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("queue", cfg.QueueKey).
		Int("workers", cfg.WorkerCount).
		Msg("application initialized")

	return &App{
		cfg:        cfg,
		redis:      rdb,
		subscriber: initRedisSubscriber(rdb),
		handler:    initImportHandler(db, rdb),
	}, nil
}

func initImportHandler(db *gorm.DB, rdb *redis.Client) bookmarkHandler.Handler {
	repo := bookmarkRepo.NewRepository(db)
	cache := cacheRepo.NewRedis(rdb)
	svc := bookmarkSvc.NewService(repo, cache)
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

	pool := worker.NewPool(a.subscriber, a.handler, a.cfg.QueueKey, a.cfg.WorkerCount, a.cfg.JobBufferSize, a.cfg.PollInterval)
	return pool.Run(ctx)
}
