package bookmark

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/huypham67/bookmark-common/pkg/tracing"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog/log"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
)

// Handle decodes the raw payload into an import message and dispatches it to the service.
// A decode failure is returned immediately so the caller can log and drop the malformed job.
func (h *handler) Handle(ctx context.Context, payload []byte) error {
	var msg bookmarkDTO.BookmarkImportMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("decode import message: %w", err)
	}

	ctx, done := tracing.Accept(ctx, h.nrApp, "job/bookmark.import", msg.TraceMetadata)
	defer done()

	if txn := newrelic.FromContext(ctx); txn != nil {
		txn.AddAttribute("job.id", msg.JobID)
		txn.AddAttribute("user.id", msg.UserID)
		txn.AddAttribute("records.count", len(msg.Records))
	}

	log.Info().
		Str("job_id", msg.JobID).
		Str("user_id", msg.UserID).
		Int("records", len(msg.Records)).
		Msg("processing import job")

	return h.service.Import(ctx, msg)
}
