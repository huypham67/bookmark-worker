package bookmark

import (
	"testing"

	"github.com/huypham67/bookmark-worker/internal/test/fixtures"
	"gorm.io/gorm"
)

func newTestRepository(t *testing.T) (Repository, *gorm.DB) {
	t.Helper()

	testDB := fixtures.NewTestDB(t, &fixtures.BookmarkTestDB{})
	return NewRepository(testDB), testDB
}
