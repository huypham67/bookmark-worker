package bookmark

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
	svcMocks "github.com/huypham67/bookmark-worker/internal/service/bookmark/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*handler, *svcMocks.Service) {
	t.Helper()
	svc := svcMocks.NewService(t)
	return NewHandler(svc, nil), svc
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	validMsg := bookmarkDTO.BookmarkImportMessage{
		JobID:  "job-1",
		UserID: "user-1",
		Records: []bookmarkDTO.BookmarkCSVRecord{
			{Description: "Go", URL: "https://go.dev"},
		},
	}
	validPayload, _ := json.Marshal(validMsg)
	errSvc := errors.New("import failed")

	testCases := []struct {
		name    string
		payload []byte
		setup   func(context.Context, *svcMocks.Service)
		verify  func(*testing.T, error)
	}{
		{
			name:    "invalid JSON → decode error, service not called",
			payload: []byte("not json"),
			setup:   func(_ context.Context, _ *svcMocks.Service) {},
			verify: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "decode import message")
			},
		},
		{
			name:    "valid payload → service called with decoded message",
			payload: validPayload,
			setup: func(ctx context.Context, svc *svcMocks.Service) {
				svc.On("Import", ctx, validMsg).Return(nil)
			},
			verify: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "service error → propagated to caller",
			payload: validPayload,
			setup: func(ctx context.Context, svc *svcMocks.Service) {
				svc.On("Import", ctx, validMsg).Return(errSvc)
			},
			verify: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, errSvc)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			h, svc := newTestHandler(t)
			tc.setup(ctx, svc)

			err := h.Handle(ctx, tc.payload)

			tc.verify(t, err)
		})
	}
}
