package fixtures

import (
	"testing"

	"github.com/huypham67/bookmark-common/pkg/sqldb"
	"gorm.io/gorm"
)

// TestDatabase is the contract implemented by per-table test DB fixtures.
type TestDatabase interface {
	SetupDB(db *gorm.DB)
	MigrateDB() error
	SeedData() error
	GetDB() *gorm.DB
}

type baseTestDB struct {
	db *gorm.DB
}

func (b *baseTestDB) SetupDB(db *gorm.DB) { b.db = db }
func (b *baseTestDB) GetDB() *gorm.DB     { return b.db }

// NewTestDB creates an in-memory SQLite DB, runs the fixture's migrations, and seeds initial data.
func NewTestDB(t *testing.T, testdb TestDatabase) *gorm.DB {
	t.Helper()

	testdb.SetupDB(sqldb.NewMock(t))

	if err := testdb.MigrateDB(); err != nil {
		t.Fatal("failed to migrate database:", err)
	}

	if err := testdb.SeedData(); err != nil {
		t.Fatal("failed to seed data:", err)
	}

	return testdb.GetDB()
}
