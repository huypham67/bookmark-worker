package bookmark

import (
	"context"

	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
)

// Handler processes a single raw job payload dequeued from the queue. It owns
// decoding the payload and dispatching it to the import service.
type Handler interface {
	Handle(ctx context.Context, payload []byte) error
}

type handler struct {
	service bookmarkSvc.Service
}

// NewHandler creates the bookmark import job handler.
func NewHandler(service bookmarkSvc.Service) Handler {
	return &handler{
		service: service,
	}
}
