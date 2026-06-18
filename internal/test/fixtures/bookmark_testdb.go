package fixtures

import (
	"time"

	"github.com/huypham67/bookmark-worker/internal/model"
	"gorm.io/gorm"
)

const (
	TestUserID1 = "user-uuid-1"
	TestUserID2 = "user-uuid-2"
)

var baseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// BookmarkTestDB is a test fixture for the bookmarks table.
type BookmarkTestDB struct {
	baseTestDB
}

func (b *BookmarkTestDB) MigrateDB() error {
	return b.db.AutoMigrate(&model.Bookmark{})
}

func (b *BookmarkTestDB) SeedData() error {
	bookmarks := []*model.Bookmark{
		{
			BaseModel:   model.BaseModel{ID: "bookmark-1-1", CreatedAt: baseTime},
			Description: "Seed Bookmark 1",
			URL:         "https://example.com/1",
			Code:        "code1001",
			UserID:      TestUserID1,
		},
		{
			BaseModel:   model.BaseModel{ID: "bookmark-1-2", CreatedAt: baseTime.Add(time.Second)},
			Description: "Seed Bookmark 2",
			URL:         "https://example.com/2",
			Code:        "code1002",
			UserID:      TestUserID1,
		},
	}

	return b.db.Session(&gorm.Session{SkipHooks: true}).CreateInBatches(bookmarks, 10).Error
}
