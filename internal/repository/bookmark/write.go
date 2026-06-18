package bookmark

import (
	"context"

	"github.com/huypham67/bookmark-common/pkg/dbutils"
	"github.com/huypham67/bookmark-common/pkg/shortcode"
	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	"github.com/huypham67/bookmark-worker/internal/model"
	"gorm.io/gorm"
)

// SaveBookmarks inserts the records in a single transaction using two-phase insert:
// batch-create first to obtain auto-increment CodeInt values, then update Code with the encoded shortcode.
func (r *repository) SaveBookmarks(ctx context.Context, userID string, records []bookmarkDTO.BookmarkCSVRecord) error {
	if len(records) == 0 {
		return nil
	}

	bookmarks := make([]*model.Bookmark, len(records))
	for i, rec := range records {
		bookmarks[i] = &model.Bookmark{
			Description: rec.Description,
			URL:         rec.URL,
			UserID:      userID,
		}
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(bookmarks, len(bookmarks)).Error; err != nil {
			return dbutils.ClassifyError(err)
		}

		for _, bm := range bookmarks {
			code, err := shortcode.EncodeSQLCode(uint64(bm.CodeInt))
			if err != nil {
				return err
			}
			bm.Code = code
			if err := tx.Model(bm).Update("code", code).Error; err != nil {
				return dbutils.ClassifyError(err)
			}
		}

		return nil
	})
}
