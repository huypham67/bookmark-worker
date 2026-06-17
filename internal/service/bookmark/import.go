package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
)

// Import currently just logs the message and delegates to the repo. Real
// validation/dedup/batching logic will be added later.
func (s *service) Import(ctx context.Context, msg bookmarkDTO.BookmarkImportMessage) error {
	return s.repo.SaveBookmarks(ctx, msg.UserID, msg.Records)
}
