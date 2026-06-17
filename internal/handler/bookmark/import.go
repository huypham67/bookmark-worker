package bookmark

import (
	"context"
	"encoding/json"
	"fmt"

	bookmarkDTO "github.com/huypham67/bookmark-worker/internal/dto/bookmark"
)

// Handle decodes the raw payload into an import message and hands it to the
// service. A decode failure is returned so the caller can log/drop the job.
func (h *handler) Handle(ctx context.Context, payload []byte) error {
	var msg bookmarkDTO.BookmarkImportMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("decode import message: %w", err)
	}

	return h.service.Import(ctx, msg)
}
