package queue

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

type redisSubscriber struct {
	client *redis.Client
}

// NewRedisSubscriber returns a new Subscriber that uses the provided Redis client to dequeue jobs from a named queue.
func NewRedisSubscriber(client *redis.Client) Subscriber {
	return &redisSubscriber{
		client: client,
	}
}

// Dequeue pops the next payload off the tail of the specified queue. It returns
// ErrQueueEmpty when the queue is empty (RPOP is non-blocking), leaving the
// polling cadence to the caller.
func (c *redisSubscriber) Dequeue(ctx context.Context, queue string) ([]byte, error) {
	raw, err := c.client.RPop(ctx, queue).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrQueueEmpty
		}
		return nil, err
	}

	return raw, nil
}
