package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
)

// Service imports a batch of bookmarks delivered as a queue message.
//
//go:generate mockery --name=Service --output=./mocks --outpkg=mocks --filename=mock_service.go
type Service interface {
	Import(ctx context.Context, msg bookmarkDTO.BookmarkImportMessage) error
}

type service struct {
	repo bookmarkRepo.Repository
}

// NewService creates the bookmark import service.
func NewService(repo bookmarkRepo.Repository) Service {
	return &service{
		repo: repo,
	}
}
