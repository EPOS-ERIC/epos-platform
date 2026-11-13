package handlers

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/internal/authcontext"
)

func (h *ogenHandler) TriggerMetadataSync(ctx context.Context) error {
	user, authenticated := authcontext.UserFromContext(ctx)

	return h.metadataSync.Trigger(user, authenticated)
}
