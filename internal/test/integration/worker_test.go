package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	pkgRedis "github.com/huypham67/bookmark-common/pkg/redis"
	"github.com/huypham67/bookmark-common/pkg/shortcode"
	"github.com/huypham67/bookmark-common/pkg/sqldb"
	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	bookmarkHandler "github.com/huypham67/bookmark-worker/internal/handler/bookmark"
	"github.com/huypham67/bookmark-worker/internal/model"
	bookmarkRepo "github.com/huypham67/bookmark-worker/internal/repository/bookmark"
	cacheRepo "github.com/huypham67/bookmark-worker/internal/repository/cache"
	"github.com/huypham67/bookmark-worker/internal/repository/queue"
	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
	"github.com/huypham67/bookmark-worker/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const queueKey = "bookmark:import:jobs"

// countingHandler wraps a real Handler and counts down wg after each job completes.
type countingHandler struct {
	inner bookmarkHandler.Handler
	wg    *sync.WaitGroup
}

func (h *countingHandler) Handle(ctx context.Context, payload []byte) error {
	defer h.wg.Done()
	return h.inner.Handle(ctx, payload)
}

type testWorker struct {
	pool  *worker.Pool
	rdb   *pkgRedis.Mock
	query func(dest any)
	wg    *sync.WaitGroup
}

func newTestWorker(t *testing.T, jobCount int) *testWorker {
	t.Helper()

	rdb := pkgRedis.NewMock(t)
	db := sqldb.NewMock(t)
	require.NoError(t, db.AutoMigrate(&model.Bookmark{}))

	var wg sync.WaitGroup
	wg.Add(jobCount)

	inner := bookmarkHandler.NewHandler(
		bookmarkSvc.NewService(
			bookmarkRepo.NewRepository(db),
			cacheRepo.NewRedis(rdb.Client),
		),
	)

	h := &countingHandler{inner: inner, wg: &wg}
	sub := queue.NewRedisSubscriber(rdb.Client)
	pool := worker.NewPool(sub, h, queueKey, 1, 10, time.Millisecond)

	return &testWorker{
		pool:  pool,
		rdb:   rdb,
		query: func(dest any) { db.Find(dest) },
		wg:    &wg,
	}
}

func (e *testWorker) push(t *testing.T, msg bookmarkDTO.BookmarkImportMessage) {
	t.Helper()
	payload, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, e.rdb.Client.LPush(context.Background(), queueKey, payload).Err())
}

// run starts the pool in a background goroutine and returns a cancel func.
// The pool stops when cancel is called and all in-flight jobs are drained.
func (e *testWorker) run(t *testing.T) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go e.pool.Run(ctx)
	return cancel
}

// waitJobs blocks until all expected jobs complete or the timeout is exceeded.
func (e *testWorker) waitJobs(t *testing.T, timeout time.Duration) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("timeout: worker did not process all jobs in time")
	}
}

func TestWorkerIntegration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		msgs   []bookmarkDTO.BookmarkImportMessage
		verify func(*testing.T, *testWorker, []model.Bookmark)
	}{
		{
			name: "single job → bookmarks saved with valid shortcodes",
			msgs: []bookmarkDTO.BookmarkImportMessage{
				{
					JobID:  "job-1",
					UserID: "user-1",
					Records: []bookmarkDTO.BookmarkCSVRecord{
						{Description: "Go", URL: "https://go.dev"},
						{Description: "GitHub", URL: "https://github.com"},
					},
				},
			},
			verify: func(t *testing.T, _ *testWorker, saved []model.Bookmark) {
				require.Len(t, saved, 2)
				for _, bm := range saved {
					assert.NotEmpty(t, bm.Code)
					assert.Equal(t, shortcode.StoreSQL, shortcode.Classify(bm.Code))
					assert.Equal(t, "user-1", bm.UserID)
				}
			},
		},
		{
			name: "multiple jobs → all records saved",
			msgs: []bookmarkDTO.BookmarkImportMessage{
				{
					JobID:  "job-1",
					UserID: "user-1",
					Records: []bookmarkDTO.BookmarkCSVRecord{
						{Description: "Go", URL: "https://go.dev"},
					},
				},
				{
					JobID:  "job-2",
					UserID: "user-1",
					Records: []bookmarkDTO.BookmarkCSVRecord{
						{Description: "GitHub", URL: "https://github.com"},
						{Description: "Rust", URL: "https://rust-lang.org"},
					},
				},
			},
			verify: func(t *testing.T, _ *testWorker, saved []model.Bookmark) {
				require.Len(t, saved, 3)
				for _, bm := range saved {
					assert.Equal(t, shortcode.StoreSQL, shortcode.Classify(bm.Code))
				}
			},
		},
		{
			name: "cache key invalidated after import",
			msgs: []bookmarkDTO.BookmarkImportMessage{
				{
					JobID:  "job-1",
					UserID: "user-1",
					Records: []bookmarkDTO.BookmarkCSVRecord{
						{Description: "Go", URL: "https://go.dev"},
					},
				},
			},
			verify: func(t *testing.T, env *testWorker, saved []model.Bookmark) {
				require.Len(t, saved, 1)
				exists, err := env.rdb.Client.Exists(context.Background(), "bookmarks:user-1").Result()
				require.NoError(t, err)
				assert.Equal(t, int64(0), exists, "cache key must be deleted after import")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tw := newTestWorker(t, len(tc.msgs))

			if tc.name == "cache key invalidated after import" {
				require.NoError(t, tw.rdb.Client.Set(context.Background(), "bookmarks:user-1", "stale", 0).Err())
			}

			for _, msg := range tc.msgs {
				tw.push(t, msg)
			}

			cancel := tw.run(t)
			tw.waitJobs(t, 5*time.Second)
			cancel()

			var saved []model.Bookmark
			tw.query(&saved)
			tc.verify(t, tw, saved)
		})
	}
}
