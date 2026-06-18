package bookmark

import (
	"context"
	"errors"
	"fmt"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
	cacheRepo "github.com/huypham67/bookmark-worker/internal/repository/cache"
)

const cacheNamespace = "bookmarks"

var ErrInternalServerError = errors.New("internal server error")

// Service processes bookmark import jobs delivered from the queue.
//
//go:generate mockery --name=Service --output=./mocks --outpkg=mocks --filename=mock_service.go
type Service interface {
	Import(ctx context.Context, msg bookmarkDTO.BookmarkImportMessage) error
}

type service struct {
	repo      bookmarkRepo.Repository
	cacheRepo cacheRepo.Repository
}

// NewService returns a Service wired with the given repository and cache.
func NewService(repo bookmarkRepo.Repository, cacheRepo cacheRepo.Repository) Service {
	return &service{
		repo:      repo,
		cacheRepo: cacheRepo,
	}
}

func buildUserCacheKey(userID string) string {
	return fmt.Sprintf("%s:%s", cacheNamespace, userID)
}
