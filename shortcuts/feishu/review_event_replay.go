package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReviewEventReplayRequest struct {
	EventID   string
	RequestID string
	Reason    string
	Confirmed bool
}

func (s *SQLiteReviewGatewayStore) ReplayReviewEvent(
	ctx context.Context, request ReviewEventReplayRequest, now time.Time,
) (ReviewEventInboxInsertResult, error) {
	request.EventID = strings.TrimSpace(request.EventID)
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.Reason = strings.TrimSpace(request.Reason)
	if request.EventID == "" || request.RequestID == "" {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Replay requires event ID and explicit request ID")
	}
	if request.Reason == "" || !request.Confirmed {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Replay requires a reason and explicit confirmation")
	}
	requestKey := stableKey("review-event-replay-request", request.RequestID)
	if existing, err := scanReviewEventInbox(s.db.QueryRowContext(ctx, reviewEventInboxSelect+` WHERE replay_request_key=?`, requestKey)); err == nil {
		return ReviewEventInboxInsertResult{Record: existing, Duplicate: true}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ReviewEventInboxInsertResult{}, err
	}
	original, err := s.GetReviewEventInbox(ctx, request.EventID)
	if err != nil {
		return ReviewEventInboxInsertResult{}, err
	}
	if original.CanonicalEventJSON == "" || original.Status == "ignored" {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Review event has no replayable canonical event")
	}
	var event NormalizedReviewEvent
	if err := json.Unmarshal([]byte(original.CanonicalEventJSON), &event); err != nil {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("decode canonical Review event: %w", err)
	}
	var installationEnabled int
	if err := s.db.QueryRowContext(ctx, `SELECT enabled FROM gitlink_installations WHERE installation_id=?`, event.InstallationID).Scan(&installationEnabled); err != nil || installationEnabled == 0 {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Replay cannot bypass a disabled installation")
	}
	var authorized, bound int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM installation_repositories
		WHERE installation_id=? AND repository=?`, event.InstallationID, event.Repository).Scan(&authorized); err != nil || authorized == 0 {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Replay cannot bypass repository authorization")
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_repository_bindings
		WHERE installation_id=? AND repository=? AND enabled=1`, event.InstallationID, event.Repository).Scan(&bound); err != nil || bound == 0 {
		return ReviewEventInboxInsertResult{}, fmt.Errorf("Replay cannot bypass repository binding")
	}
	event.Source = ReviewEventSourceManualReplay
	event.Synthetic = true
	event.ReplayOf = original.Event.EventID
	event.ReceivedAt = now.UTC().Format(time.RFC3339Nano)
	event.DeliveryKey = reviewEventSyntheticDeliveryKey(event.Source, event.InstallationID, requestKey)
	event.DeliveryHash = reviewResourceIdentifierHash(requestKey)
	event.EventID = reviewEventID(event)
	return s.InsertReviewEventInbox(ctx, ReviewEventNormalization{Event: event, Status: "normalized"}, ReviewEventIngressMetadata{
		SignatureStatus: "replay_not_reverified", TimestampStatus: "replay_not_reverified",
		DeliveryStatus: "synthetic", ReplayRequestKey: requestKey,
	})
}
