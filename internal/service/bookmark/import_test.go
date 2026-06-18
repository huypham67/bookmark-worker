package bookmark

import (
	"context"
	"errors"
	"testing"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	bookmarkMocks "github.com/huypham67/bookmark-worker/internal/repository/bookmark/mocks"
	cacheMocks "github.com/huypham67/bookmark-worker/internal/repository/cache/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (Service, *bookmarkMocks.Repository, *cacheMocks.Repository) {
	t.Helper()
	repo := bookmarkMocks.NewRepository(t)
	cache := cacheMocks.NewRepository(t)
	return NewService(repo, cache), repo, cache
}

func TestService_Import(t *testing.T) {
	t.Parallel()

	msg := bookmarkDTO.BookmarkImportMessage{
		JobID:  "job-1",
		UserID: "user-1",
		Records: []bookmarkDTO.BookmarkCSVRecord{
			{Description: "Go", URL: "https://go.dev"},
			{Description: "GitHub", URL: "https://github.com"},
		},
	}

	errRedis := errors.New("redis down")
	errDB := errors.New("db unavailable")

	testCases := []struct {
		name   string
		setup  func(context.Context, *cacheMocks.Repository, *bookmarkMocks.Repository)
		verify func(*testing.T, error)
	}{
		{
			name: "cache invalidation fails → abort without saving",
			setup: func(ctx context.Context, cache *cacheMocks.Repository, _ *bookmarkMocks.Repository) {
				cache.On("DeleteCacheByHashKey", ctx, "bookmarks:user-1").Return(errRedis)
			},
			verify: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ErrInternalServerError)
			},
		},
		{
			name: "save fails → repo error propagated",
			setup: func(ctx context.Context, cache *cacheMocks.Repository, repo *bookmarkMocks.Repository) {
				cache.On("DeleteCacheByHashKey", ctx, "bookmarks:user-1").Return(nil)
				repo.On("SaveBookmarks", ctx, "user-1", msg.Records).Return(errDB)
			},
			verify: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errDB)
			},
		},
		{
			name: "success → nil",
			setup: func(ctx context.Context, cache *cacheMocks.Repository, repo *bookmarkMocks.Repository) {
				cache.On("DeleteCacheByHashKey", ctx, "bookmarks:user-1").Return(nil)
				repo.On("SaveBookmarks", ctx, "user-1", msg.Records).Return(nil)
			},
			verify: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, repo, cache := newTestService(t)
			tc.setup(ctx, cache, repo)

			err := svc.Import(ctx, msg)

			tc.verify(t, err)
		})
	}
}
