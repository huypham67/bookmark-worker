package bookmark

import (
	"context"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	"github.com/rs/zerolog/log"
)

// Import invalidates the user's bookmark cache then bulk-inserts the records.
// Cache invalidation runs first; if it fails the import is aborted to avoid serving stale data after a successful write.
func (s *service) Import(ctx context.Context, msg bookmarkDTO.BookmarkImportMessage) error {
	hashKey := buildUserCacheKey(msg.UserID)
	if err := s.cacheRepo.DeleteCacheByHashKey(ctx, hashKey); err != nil {
		log.Error().
			Err(err).
			Str("job_id", msg.JobID).
			Str("user_id", msg.UserID).
			Msg("failed to invalidate user cache; aborting import")
		return ErrInternalServerError
	}

	if err := s.repo.SaveBookmarks(ctx, msg.UserID, msg.Records); err != nil {
		log.Error().
			Err(err).
			Str("job_id", msg.JobID).
			Str("user_id", msg.UserID).
			Int("records", len(msg.Records)).
			Msg("failed to save bookmarks")
		return err
	}

	log.Info().
		Str("job_id", msg.JobID).
		Str("user_id", msg.UserID).
		Int("records", len(msg.Records)).
		Msg("import job completed")

	return nil
}
