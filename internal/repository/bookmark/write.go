package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
)

// SaveBookmarks is a stub: it logs each record instead of writing to the DB.
// Real persistence logic will replace this body later.
func (r *repository) SaveBookmarks(_ context.Context, userID string, records []bookmarkDTO.BookmarkCSVRecord) error {
	return nil
}
