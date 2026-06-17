package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
)

// Repository persists imported bookmarks.
//
//go:generate mockery --name=Repository --output=./mocks --outpkg=mocks --filename=mock_repo.go
type Repository interface {
	SaveBookmarks(ctx context.Context, userID string, records []bookmarkDTO.BookmarkCSVRecord) error
}

type repository struct{}

// NewRepository creates the bookmark repository. The DB-backed implementation
// is not wired yet; for now it only logs what it would persist.
func NewRepository() Repository {
	return &repository{}
}
