package bootstrap

import (
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds runtime configuration for the worker, loaded from environment variables.
type Config struct {
	ServiceName   string        `envconfig:"SERVICE_NAME" required:"true"`
	QueueKey      string        `envconfig:"QUEUE_KEY" default:"bookmark:import:jobs"`
	WorkerCount   int           `envconfig:"WORKER_COUNT" default:"5"`
	JobBufferSize int           `envconfig:"JOB_BUFFER_SIZE" default:"100"`
	PollInterval  time.Duration `envconfig:"POLL_INTERVAL" default:"1s"`
}

// NewConfig reads and validates worker configuration from environment variables.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	cfg.ServiceName = strings.TrimSpace(cfg.ServiceName)

	return cfg, nil
}
