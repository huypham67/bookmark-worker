package bookmark

import (
	"context"

	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
)

// Handler decodes a raw queue payload and dispatches it to the import service.
//
//go:generate mockery --name=Handler --output=./mocks --outpkg=mocks --filename=mock_handler.go
type Handler interface {
	Handle(ctx context.Context, payload []byte) error
}

type handler struct {
	service bookmarkSvc.Service
}

// NewHandler returns a Handler backed by the given service.
func NewHandler(service bookmarkSvc.Service) Handler {
	return &handler{
		service: service,
	}
}
