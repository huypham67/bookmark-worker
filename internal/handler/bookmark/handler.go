package bookmark

import (
	"github.com/newrelic/go-agent/v3/newrelic"

	bookmarkSvc "github.com/huypham67/bookmark-worker/internal/service/bookmark"
)

type handler struct {
	service bookmarkSvc.Service
	nrApp   *newrelic.Application
}

// NewHandler returns a handler backed by the given service.
func NewHandler(service bookmarkSvc.Service, nrApp *newrelic.Application) *handler {
	return &handler{
		service: service,
		nrApp:   nrApp,
	}
}
