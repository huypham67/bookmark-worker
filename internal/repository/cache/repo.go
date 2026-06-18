package cache

import "context"

// Repository provides cache invalidation operations used by the worker.
//
//go:generate mockery --name=Repository --output=./mocks --outpkg=mocks --filename=mock_repo.go
type Repository interface {
	DeleteCacheByHashKey(ctx context.Context, hashKey string) error
}
