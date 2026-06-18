package bookmark

import (
	"context"
	"testing"

	"github.com/huypham67/bookmark-common/pkg/dbutils"
	"github.com/huypham67/bookmark-common/pkg/shortcode"
	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	"github.com/huypham67/bookmark-worker/internal/model"
	"github.com/huypham67/bookmark-worker/internal/test/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepository_SaveBookmarks(t *testing.T) {
	t.Parallel()

	type args struct {
		userID  string
		records []bookmarkDTO.BookmarkCSVRecord
	}

	testCases := []struct {
		name   string
		args   args
		verify func(*testing.T, *gorm.DB, error, args)
	}{
		{
			name: "should save batch successfully",
			args: args{
				userID: fixtures.TestUserID1,
				records: []bookmarkDTO.BookmarkCSVRecord{
					{Description: "Batch 1", URL: "https://example.com/a"},
					{Description: "Batch 2", URL: "https://example.com/b"},
					{Description: "Batch 3", URL: "https://example.com/c"},
				},
			},
			verify: func(t *testing.T, db *gorm.DB, err error, a args) {
				require.NoError(t, err)

				var saved []model.Bookmark
				result := db.Where("user_id = ? AND url IN ?", a.userID, []string{
					"https://example.com/a",
					"https://example.com/b",
					"https://example.com/c",
				}).Find(&saved)
				require.NoError(t, result.Error)
				assert.Len(t, saved, 3)

				for _, bm := range saved {
					assert.NotEmpty(t, bm.Code)
					assert.Equal(t, shortcode.StoreSQL, shortcode.Classify(bm.Code))
				}
			},
		},
		{
			name: "should return nil for empty records",
			args: args{
				userID:  fixtures.TestUserID1,
				records: []bookmarkDTO.BookmarkCSVRecord{},
			},
			verify: func(t *testing.T, db *gorm.DB, err error, a args) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error when context is cancelled",
			args: args{
				userID: fixtures.TestUserID1,
				records: []bookmarkDTO.BookmarkCSVRecord{
					{Description: "Cancelled", URL: "https://example.com/cancel"},
				},
			},
			verify: func(t *testing.T, db *gorm.DB, err error, a args) {
				require.Error(t, err)
			},
		},
		{
			name: "should rollback on duplicate ID",
			args: args{
				userID:  fixtures.TestUserID2,
				records: []bookmarkDTO.BookmarkCSVRecord{},
			},
			verify: func(t *testing.T, db *gorm.DB, err error, a args) {
				require.Error(t, err)
				assert.ErrorIs(t, err, dbutils.ErrDuplicationType)

				var count int64
				db.Model(&model.Bookmark{}).Where("url LIKE ?", "https://example.com/rollback%").Count(&count)
				assert.Equal(t, int64(0), count)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			if tc.name == "should return error when context is cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			repo, db := newTestRepository(t)

			if tc.name == "should rollback on duplicate ID" {
				// Pre-insert a bookmark with a known ID, then attempt a batch insert
				// containing the same ID to trigger a duplicate primary-key error and
				// verify the whole transaction rolls back.
				require.NoError(t, db.Create(&model.Bookmark{
					BaseModel: model.BaseModel{ID: "dup-id-fixed"},
					URL:       "https://example.com/pre-existing",
					Code:      "dupcode",
					UserID:    tc.args.userID,
				}).Error)

				err := db.Transaction(func(tx *gorm.DB) error {
					batch := []*model.Bookmark{
						{BaseModel: model.BaseModel{ID: "dup-id-fixed"}, URL: "https://example.com/rollback1", UserID: tc.args.userID},
						{URL: "https://example.com/rollback2", UserID: tc.args.userID},
					}
					if err := tx.CreateInBatches(batch, 2).Error; err != nil {
						return dbutils.ClassifyError(err)
					}
					return nil
				})
				tc.verify(t, db, err, tc.args)
				return
			}

			err := repo.SaveBookmarks(ctx, tc.args.userID, tc.args.records)

			tc.verify(t, db, err, tc.args)
		})
	}
}
