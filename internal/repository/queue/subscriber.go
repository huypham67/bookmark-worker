package queue

import (
	"context"
	"errors"
)

var (
	ErrQueueEmpty = errors.New("queue is empty")
)

// Subscriber reads jobs from a named queue.
//
//go:generate mockery --name=Subscriber --output=./mocks --outpkg=mocks --filename=mock_subscriber.go
type Subscriber interface {
	Dequeue(ctx context.Context, queue string) ([]byte, error)
}
