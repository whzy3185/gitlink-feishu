package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ReviewPreparedJobEnqueuer interface {
	EnqueuePreparedJob(context.Context, ReviewGatewayJob) (bool, error)
}

type ReviewPreparedJobAdmissionEnqueuer interface {
	EnqueuePreparedJobWithAdmission(context.Context, ReviewGatewayJob, string) (ReviewJobAdmissionResult, error)
}

type ReviewEventRoute struct {
	EventID              string `json:"event_id"`
	SubscriptionID       string `json:"subscription_id"`
	InstallationID       string `json:"installation_id"`
	ChatIDHash           string `json:"chat_id_hash"`
	Repository           string `json:"repository"`
	PRNumber             int    `json:"pr_number"`
	SubscriptionRevision int    `json:"subscription_revision"`
	NotificationMode     string `json:"notification_mode"`
	JobID                string `json:"job_id,omitempty"`
	RouteStatus          string `json:"route_status"`
	ReasonCode           string `json:"reason_code,omitempty"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	chatID               string
}

type ReviewEventInboxProcessor struct {
	Store         *SQLiteReviewGatewayStore
	Queue         ReviewPreparedJobEnqueuer
	LeaseOwner    string
	LeaseDuration time.Duration
	Now           func() time.Time
	wake          chan struct{}
}

func NewReviewEventInboxProcessor(store *SQLiteReviewGatewayStore, queue ReviewPreparedJobEnqueuer, leaseOwner string) *ReviewEventInboxProcessor {
	return &ReviewEventInboxProcessor{
		Store: store, Queue: queue, LeaseOwner: leaseOwner,
		LeaseDuration: 2 * time.Minute, Now: time.Now, wake: make(chan struct{}, 1),
	}
}

func (p *ReviewEventInboxProcessor) Wake() {
	if p == nil || p.wake == nil {
		return
	}
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *ReviewEventInboxProcessor) Run(ctx context.Context) {
	if p == nil {
		return
	}
	if p.wake == nil {
		p.wake = make(chan struct{}, 1)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	p.Wake()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-p.wake:
		}
		for {
			processed, err := p.ProcessOne(ctx)
			if err != nil || !processed {
				break
			}
		}
	}
}

func (p *ReviewEventInboxProcessor) ProcessOne(ctx context.Context) (bool, error) {
	if p == nil || p.Store == nil || p.Queue == nil {
		return false, fmt.Errorf("Review event processor store and queue are required")
	}
	now := time.Now().UTC()
	if p.Now != nil {
		now = p.Now().UTC()
	}
	owner := strings.TrimSpace(p.LeaseOwner)
	if owner == "" {
		owner = "review-event-processor"
	}
	record, err := p.Store.ClaimReviewEventInbox(ctx, ReviewEventClaimOptions{
		LeaseOwner: owner, Now: now, LeaseDuration: p.LeaseDuration,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := validateNormalizedReviewEvent(record.Event); err != nil {
		_ = p.Store.finishReviewEventInbox(ctx, record.Event.EventID, owner, "failed", "invalid_canonical_event", err.Error(), now)
		return true, err
	}
	subscriptions, err := p.Store.listReviewEventRoutesForEvent(ctx, record.Event)
	if err != nil {
		_ = p.Store.retryReviewEventInbox(ctx, record, owner, err, now)
		return true, err
	}
	if len(subscriptions) == 0 {
		err = p.Store.finishReviewEventInbox(ctx, record.Event.EventID, owner, "processed", "no_active_subscription", "", now)
		return true, err
	}
	for _, subscription := range subscriptions {
		route, created, err := p.Store.ensureReviewEventRoute(ctx, record.Event, subscription, now)
		if err != nil {
			_ = p.Store.retryReviewEventInbox(ctx, record, owner, err, now)
			return true, err
		}
		if !created && (route.RouteStatus == "queued" || route.RouteStatus == "coalesced" || route.RouteStatus == "duplicate") {
			continue
		}
		job := buildReviewEventRouteJob(record.Event, subscription, now)
		saved := false
		admission := ReviewJobAdmissionResult{}
		var enqueueErr error
		if admissionQueue, ok := p.Queue.(ReviewPreparedJobAdmissionEnqueuer); ok {
			admission, enqueueErr = admissionQueue.EnqueuePreparedJobWithAdmission(ctx, job, "event_route")
			saved = admission.Created
		} else {
			saved, enqueueErr = p.Queue.EnqueuePreparedJob(ctx, job)
			admission = ReviewJobAdmissionResult{Created: saved, JobID: job.JobID}
		}
		status := "queued"
		reason := ""
		if enqueueErr != nil {
			status, reason = "failed", "enqueue_failed"
		} else if admission.Coalesced {
			status = "coalesced"
		} else if !saved {
			status = "duplicate"
		}
		if updateErr := p.Store.updateReviewEventRoute(ctx, route, firstNonEmpty(admission.JobID, job.JobID), status, reason, now); updateErr != nil {
			_ = p.Store.retryReviewEventInbox(ctx, record, owner, updateErr, now)
			return true, updateErr
		}
		if enqueueErr != nil {
			_ = p.Store.retryReviewEventInbox(ctx, record, owner, enqueueErr, now)
			return true, enqueueErr
		}
	}
	if _, err := p.Store.db.ExecContext(ctx, `UPDATE review_event_inbox
		SET status='routed', route_revision=route_revision+1 WHERE event_id=? AND lease_owner=?`,
		record.Event.EventID, owner); err != nil {
		_ = p.Store.retryReviewEventInbox(ctx, record, owner, err, now)
		return true, err
	}
	return true, p.Store.finishReviewEventInbox(ctx, record.Event.EventID, owner, "processed", "", "", now)
}

func (s *SQLiteReviewGatewayStore) listReviewEventRoutesForEvent(
	ctx context.Context, event NormalizedReviewEvent,
) ([]ReviewChatSubscription, error) {
	groupColumn := reviewEventSubscriptionColumn(event.Action)
	if groupColumn == "" {
		return nil, nil
	}
	query := `SELECT s.subscription_id, s.installation_id, s.chat_id, s.repository,
		s.pull_events, s.review_events, s.thread_events, s.merge_events, s.ci_events,
		s.notification_mode, s.enabled, s.revision, s.created_by_hash, s.updated_by_hash,
		s.created_at, s.updated_at
		FROM review_chat_subscriptions s
		JOIN gitlink_installations i ON i.installation_id=s.installation_id AND i.enabled=1
		JOIN installation_repositories ir ON ir.installation_id=s.installation_id AND ir.repository=s.repository
		JOIN chat_repository_bindings b ON b.installation_id=s.installation_id
		 AND b.chat_id=s.chat_id AND b.repository=s.repository AND b.enabled=1
		WHERE s.installation_id=? AND s.repository=? AND s.enabled=1 AND s.` + groupColumn + `=1
		ORDER BY s.subscription_id`
	rows, err := s.db.QueryContext(ctx, query, event.InstallationID, event.Repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewChatSubscription{}
	for rows.Next() {
		var subscription ReviewChatSubscription
		var chatID string
		var pull, review, thread, merge, ci, enabled int
		if err := rows.Scan(&subscription.SubscriptionID, &subscription.InstallationID, &chatID,
			&subscription.Repository, &pull, &review, &thread, &merge, &ci,
			&subscription.NotificationMode, &enabled, &subscription.Revision,
			&subscription.CreatedByHash, &subscription.UpdatedByHash,
			&subscription.CreatedAt, &subscription.UpdatedAt); err != nil {
			return nil, err
		}
		subscription.chatID = chatID
		subscription.ChatIDHash = reviewResourceIdentifierHash(chatID)
		subscription.Enabled = enabled != 0
		subscription.EventGroups = reviewSubscriptionGroups(pull, review, thread, merge, ci)
		result = append(result, subscription)
	}
	return result, rows.Err()
}

func reviewEventSubscriptionColumn(action string) string {
	switch {
	case action == "pull_request.merged" || action == "pull_request.closed":
		return "merge_events"
	case strings.HasPrefix(action, "pull_request."):
		return "pull_events"
	case strings.HasPrefix(action, "review_thread."):
		return "thread_events"
	case strings.HasPrefix(action, "review."):
		return "review_events"
	case strings.HasPrefix(action, "ci."):
		return "ci_events"
	default:
		return ""
	}
}

func (s *SQLiteReviewGatewayStore) ensureReviewEventRoute(
	ctx context.Context, event NormalizedReviewEvent, subscription ReviewChatSubscription, now time.Time,
) (ReviewEventRoute, bool, error) {
	nowText := reviewGatewayTimestamp(now)
	result, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO review_event_routes (
		event_id, subscription_id, installation_id, chat_id, repository, pr_number,
		subscription_revision, notification_mode, route_status, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'planned', ?, ?)`, event.EventID, subscription.SubscriptionID,
		event.InstallationID, subscription.chatID, event.Repository, event.PRNumber,
		subscription.Revision, subscription.NotificationMode, nowText, nowText)
	if err != nil {
		return ReviewEventRoute{}, false, err
	}
	affected, _ := result.RowsAffected()
	route, err := s.getReviewEventRoute(ctx, event.EventID, subscription.SubscriptionID)
	return route, affected == 1, err
}

func (s *SQLiteReviewGatewayStore) getReviewEventRoute(ctx context.Context, eventID, subscriptionID string) (ReviewEventRoute, error) {
	var route ReviewEventRoute
	err := s.db.QueryRowContext(ctx, `SELECT event_id, subscription_id, installation_id, chat_id,
		repository, pr_number, subscription_revision, notification_mode, job_id, route_status,
		reason_code, created_at, updated_at FROM review_event_routes WHERE event_id=? AND subscription_id=?`,
		eventID, subscriptionID).Scan(&route.EventID, &route.SubscriptionID, &route.InstallationID,
		&route.chatID, &route.Repository, &route.PRNumber, &route.SubscriptionRevision,
		&route.NotificationMode, &route.JobID, &route.RouteStatus, &route.ReasonCode,
		&route.CreatedAt, &route.UpdatedAt)
	route.ChatIDHash = reviewResourceIdentifierHash(route.chatID)
	return route, err
}

func (s *SQLiteReviewGatewayStore) ListReviewEventRoutes(ctx context.Context, eventID string) ([]ReviewEventRoute, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id, subscription_id, installation_id, chat_id,
		repository, pr_number, subscription_revision, notification_mode, job_id, route_status,
		reason_code, created_at, updated_at FROM review_event_routes WHERE event_id=?
		ORDER BY subscription_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewEventRoute{}
	for rows.Next() {
		var route ReviewEventRoute
		if err := rows.Scan(&route.EventID, &route.SubscriptionID, &route.InstallationID,
			&route.chatID, &route.Repository, &route.PRNumber, &route.SubscriptionRevision,
			&route.NotificationMode, &route.JobID, &route.RouteStatus, &route.ReasonCode,
			&route.CreatedAt, &route.UpdatedAt); err != nil {
			return nil, err
		}
		route.ChatIDHash = reviewResourceIdentifierHash(route.chatID)
		result = append(result, route)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) updateReviewEventRoute(
	ctx context.Context, route ReviewEventRoute, jobID, status, reason string, now time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `UPDATE review_event_routes SET job_id=?, route_status=?,
		reason_code=?, updated_at=? WHERE event_id=? AND subscription_id=?`,
		jobID, status, reason, reviewGatewayTimestamp(now), route.EventID, route.SubscriptionID)
	return err
}

func buildReviewEventRouteJob(event NormalizedReviewEvent, subscription ReviewChatSubscription, now time.Time) ReviewGatewayJob {
	dedupeKey := strings.Join([]string{
		"review:event-route", event.EventID, subscription.SubscriptionID,
		fmt.Sprintf("%d", subscription.Revision),
	}, ":")
	return ReviewGatewayJob{
		SchemaVersion:    reviewGatewayJobSchema,
		JobID:            stableKey("review-event-job", dedupeKey),
		DedupeKey:        dedupeKey,
		Status:           "queued",
		Mode:             "preview",
		Action:           "refresh_review_context",
		InstallationID:   event.InstallationID,
		Repository:       event.Repository,
		PRNumber:         event.PRNumber,
		ChatID:           subscription.chatID,
		RequestedBy:      "gitlink-event:" + event.InstallationID,
		SourceEventID:    event.EventID,
		NotifyChat:       subscription.NotificationMode == ReviewNotificationCanonicalAndNotice,
		NotificationMode: subscription.NotificationMode,
		CreatedAt:        reviewGatewayTimestamp(now),
		MutatesGitLink:   false,
		MaxAttempts:      reviewGatewayDefaultMaxAttempts,
		NextAttemptAt:    reviewGatewayTimestamp(now),
	}
}
