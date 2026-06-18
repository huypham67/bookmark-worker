package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	"gorm.io/gorm"
)

// Repository persists bookmark records to the database.
//
//go:generate mockery --name=Repository --output=./mocks --outpkg=mocks --filename=mock_repo.go
type Repository interface {
	SaveBookmarks(ctx context.Context, userID string, records []bookmarkDTO.BookmarkCSVRecord) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository returns a Repository backed by the given GORM DB.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}
