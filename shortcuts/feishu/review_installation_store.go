package feishu

import (
	"context"
	"time"
)

// SyncReviewGatewayConfiguration atomically replaces the local runtime view of
// GitLink installations and chat bindings. It is retained for callers that do
// not participate in optimistic concurrency. New configuration commands should
// call ApplyReviewGatewayConfiguration and provide ExpectedRevision.
func (s *SQLiteReviewGatewayStore) SyncReviewGatewayConfiguration(
	ctx context.Context,
	bindings ReviewGatewayBindings,
	source string,
	now time.Time,
) error {
	_, _, err := s.ApplyReviewGatewayConfiguration(ctx, bindings, ReviewConfigurationApplyOptions{Source: source}, now)
	return err
}

func boolToSQLiteInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
