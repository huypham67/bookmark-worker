package bootstrap

import (
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	pkgRedis "github.com/huypham67/bookmark-common/pkg/redis"
	"github.com/huypham67/bookmark-common/pkg/sqldb"
	"github.com/huypham67/bookmark-common/pkg/tracing"
	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
	cacheRepo "github.com/huypham67/bookmark-worker/internal/repository/cache"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
	"github.com/huypham67/bookmark-worker/internal/worker"
)

// Container holds all worker dependencies.
type Container struct {
	Config     *Config
	DB         *gorm.DB
	Redis      *redis.Client
	NRApp      *newrelic.Application
	Subscriber queue.Subscriber
	Handler    worker.Handler
}

// NewContainer initializes all infrastructure clients and the job handler pipeline.
func NewContainer() (*Container, error) {
	cfg, err := NewConfig()
	if err != nil {
		return nil, err
	}

	nrApp, err := tracing.NewApplication("")
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize New Relic")
		return nil, err
	}

	rdb, err := pkgRedis.NewClient("")
	if err != nil {
		return nil, err
	}
	rdb.AddHook(nrredis.NewHook(rdb.Options()))

	db, err := sqldb.NewInstrumentedClient("")
	if err != nil {
		return nil, err
	}

	svc := bookmarkSvc.NewService(
		bookmarkRepo.NewRepository(db),
		cacheRepo.NewRedis(rdb),
	)

	return &Container{
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		NRApp:      nrApp,
		Subscriber: queue.NewRedisSubscriber(rdb),
		Handler:    bookmarkHandler.NewHandler(svc, nrApp),
	}, nil
}

// Close gracefully shuts down all resources.
func (c *Container) Close() error {
	if c.Redis != nil {
		_ = c.Redis.Close()
	}
	if c.DB != nil {
		if sqlDB, err := c.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	if c.NRApp != nil {
		c.NRApp.Shutdown(0)
	}
	return nil
}
