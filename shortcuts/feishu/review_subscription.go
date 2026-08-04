package feishu

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ReviewSubscriptionGroupPulls   = "pulls"
	ReviewSubscriptionGroupReviews = "reviews"
	ReviewSubscriptionGroupThreads = "threads"
	ReviewSubscriptionGroupMerge   = "merge"
	ReviewSubscriptionGroupCI      = "ci"

	ReviewNotificationCanonicalOnly      = "canonical_only"
	ReviewNotificationCanonicalAndNotice = "canonical_and_notice"
	ReviewNotificationSilentRefresh      = "silent_refresh"
)

var ErrReviewSubscriptionConflict = errors.New("review subscription changed concurrently")

type ReviewChatSubscription struct {
	SubscriptionID   string   `json:"subscription_id"`
	InstallationID   string   `json:"installation_id"`
	ChatIDHash       string   `json:"chat_id_hash"`
	Repository       string   `json:"repository"`
	EventGroups      []string `json:"event_groups"`
	NotificationMode string   `json:"notification_mode"`
	Enabled          bool     `json:"enabled"`
	Revision         int      `json:"revision"`
	CreatedByHash    string   `json:"created_by_hash,omitempty"`
	UpdatedByHash    string   `json:"updated_by_hash,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	chatID           string
}

type ReviewSubscriptionChange struct {
	InstallationID   string
	ChatID           string
	Repository       string
	ActorID          string
	AddGroups        []string
	RemoveGroups     []string
	NotificationMode string
	ExpectedRevision int
}

func reviewChatSubscriptionKey(installationID, chatID, repository string) string {
	return stableKey("review-chat-subscription", strings.TrimSpace(installationID),
		reviewResourceIdentifierHash(chatID), strings.TrimSpace(repository))
}

func normalizeReviewSubscriptionGroups(groups []string) ([]string, error) {
	set := map[string]bool{}
	for _, group := range groups {
		group = strings.ToLower(strings.TrimSpace(group))
		switch group {
		case ReviewSubscriptionGroupPulls, ReviewSubscriptionGroupReviews,
			ReviewSubscriptionGroupThreads, ReviewSubscriptionGroupMerge, ReviewSubscriptionGroupCI:
			set[group] = true
		case "":
		default:
			return nil, fmt.Errorf("unsupported Review event group %q", group)
		}
	}
	result := make([]string, 0, len(set))
	for group := range set {
		result = append(result, group)
	}
	sort.Strings(result)
	return result, nil
}

func validateReviewNotificationMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case ReviewNotificationCanonicalOnly, ReviewNotificationCanonicalAndNotice, ReviewNotificationSilentRefresh:
		return nil
	default:
		return fmt.Errorf("unsupported Review notification mode %q", mode)
	}
}

func (s *SQLiteReviewGatewayStore) ChangeReviewChatSubscription(
	ctx context.Context, change ReviewSubscriptionChange, now time.Time,
) (ReviewChatSubscription, error) {
	change.InstallationID = strings.TrimSpace(change.InstallationID)
	change.ChatID = strings.TrimSpace(change.ChatID)
	change.Repository = strings.TrimSpace(change.Repository)
	change.ActorID = strings.TrimSpace(change.ActorID)
	if change.InstallationID == "" || change.ChatID == "" || change.Repository == "" || change.ActorID == "" {
		return ReviewChatSubscription{}, fmt.Errorf("installation, chat, repository, and actor are required")
	}
	add, err := normalizeReviewSubscriptionGroups(change.AddGroups)
	if err != nil {
		return ReviewChatSubscription{}, err
	}
	remove, err := normalizeReviewSubscriptionGroups(change.RemoveGroups)
	if err != nil {
		return ReviewChatSubscription{}, err
	}
	if change.NotificationMode != "" {
		change.NotificationMode = strings.TrimSpace(change.NotificationMode)
		if err := validateReviewNotificationMode(change.NotificationMode); err != nil {
			return ReviewChatSubscription{}, err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewChatSubscription{}, err
	}
	defer tx.Rollback()
	if err := requireReviewSubscriptionAdministrator(ctx, tx, change); err != nil {
		return ReviewChatSubscription{}, err
	}
	current, err := readReviewChatSubscriptionTx(ctx, tx, change.InstallationID, change.ChatID, change.Repository)
	if err != nil {
		return ReviewChatSubscription{}, err
	}
	if current.SubscriptionID == "" {
		if change.ExpectedRevision > 0 {
			return ReviewChatSubscription{}, ErrReviewSubscriptionConflict
		}
		current = ReviewChatSubscription{
			SubscriptionID: reviewChatSubscriptionKey(change.InstallationID, change.ChatID, change.Repository),
			InstallationID: change.InstallationID, ChatIDHash: reviewResourceIdentifierHash(change.ChatID),
			Repository: change.Repository, NotificationMode: ReviewNotificationCanonicalOnly,
			Revision: 1, CreatedByHash: reviewResourceIdentifierHash(change.ActorID),
			CreatedAt: now.UTC().Format(time.RFC3339Nano), chatID: change.ChatID,
		}
		current.EventGroups = add
		current.Enabled = len(current.EventGroups) > 0
		if change.NotificationMode != "" {
			current.NotificationMode = change.NotificationMode
		}
		current.UpdatedByHash = reviewResourceIdentifierHash(change.ActorID)
		current.UpdatedAt = now.UTC().Format(time.RFC3339Nano)
		if _, err := tx.ExecContext(ctx, `INSERT INTO review_chat_subscriptions (
			subscription_id, installation_id, chat_id, repository,
			pull_events, review_events, thread_events, merge_events, ci_events,
			notification_mode, enabled, revision, created_by_hash, updated_by_hash,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
			current.SubscriptionID, current.InstallationID, change.ChatID, current.Repository,
			reviewSubscriptionGroupInt(current.EventGroups, ReviewSubscriptionGroupPulls),
			reviewSubscriptionGroupInt(current.EventGroups, ReviewSubscriptionGroupReviews),
			reviewSubscriptionGroupInt(current.EventGroups, ReviewSubscriptionGroupThreads),
			reviewSubscriptionGroupInt(current.EventGroups, ReviewSubscriptionGroupMerge),
			reviewSubscriptionGroupInt(current.EventGroups, ReviewSubscriptionGroupCI),
			current.NotificationMode, boolToSQLiteInteger(current.Enabled), current.CreatedByHash,
			current.UpdatedByHash, current.CreatedAt, current.UpdatedAt); err != nil {
			return ReviewChatSubscription{}, err
		}
	} else {
		if change.ExpectedRevision > 0 && current.Revision != change.ExpectedRevision {
			return ReviewChatSubscription{}, ErrReviewSubscriptionConflict
		}
		groupSet := stringSet(current.EventGroups)
		for _, group := range add {
			groupSet[group] = true
		}
		for _, group := range remove {
			delete(groupSet, group)
		}
		groups := make([]string, 0, len(groupSet))
		for group := range groupSet {
			groups = append(groups, group)
		}
		sort.Strings(groups)
		mode := current.NotificationMode
		if change.NotificationMode != "" {
			mode = change.NotificationMode
		}
		if equalReviewSubscriptionStrings(groups, current.EventGroups) && mode == current.NotificationMode {
			if err := tx.Commit(); err != nil {
				return ReviewChatSubscription{}, err
			}
			return current, nil
		}
		nowText := now.UTC().Format(time.RFC3339Nano)
		result, err := tx.ExecContext(ctx, `UPDATE review_chat_subscriptions SET
			pull_events=?, review_events=?, thread_events=?, merge_events=?, ci_events=?,
			notification_mode=?, enabled=?, revision=revision+1,
			updated_by_hash=?, updated_at=?
			WHERE subscription_id=? AND revision=?`,
			reviewSubscriptionGroupInt(groups, ReviewSubscriptionGroupPulls),
			reviewSubscriptionGroupInt(groups, ReviewSubscriptionGroupReviews),
			reviewSubscriptionGroupInt(groups, ReviewSubscriptionGroupThreads),
			reviewSubscriptionGroupInt(groups, ReviewSubscriptionGroupMerge),
			reviewSubscriptionGroupInt(groups, ReviewSubscriptionGroupCI), mode,
			boolToSQLiteInteger(len(groups) > 0), reviewResourceIdentifierHash(change.ActorID), nowText,
			current.SubscriptionID, current.Revision)
		if err != nil {
			return ReviewChatSubscription{}, err
		}
		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			return ReviewChatSubscription{}, ErrReviewSubscriptionConflict
		}
	}
	if err := tx.Commit(); err != nil {
		return ReviewChatSubscription{}, err
	}
	return s.GetReviewChatSubscription(ctx, change.InstallationID, change.ChatID, change.Repository)
}

func requireReviewSubscriptionAdministrator(ctx context.Context, tx *sql.Tx, change ReviewSubscriptionChange) error {
	var enabled int
	var adminsJSON string
	err := tx.QueryRowContext(ctx, `SELECT enabled, admin_user_ids_json
		FROM chat_repository_bindings WHERE chat_id=? AND installation_id=? AND repository=?`,
		change.ChatID, change.InstallationID, change.Repository).Scan(&enabled, &adminsJSON)
	if errors.Is(err, sql.ErrNoRows) || enabled == 0 {
		return fmt.Errorf("enabled repository binding is required")
	}
	if err != nil {
		return err
	}
	var admins []string
	if err := json.Unmarshal([]byte(adminsJSON), &admins); err != nil {
		return fmt.Errorf("parse binding administrators: %w", err)
	}
	if !containsReviewGatewayString(admins, change.ActorID) {
		return fmt.Errorf("subscription change requires a configured chat administrator")
	}
	var installationEnabled, repositoryAllowed int
	if err := tx.QueryRowContext(ctx, `SELECT enabled FROM gitlink_installations WHERE installation_id=?`, change.InstallationID).Scan(&installationEnabled); err != nil || installationEnabled == 0 {
		return fmt.Errorf("enabled GitLink installation is required")
	}
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM installation_repositories WHERE installation_id=? AND repository=?`, change.InstallationID, change.Repository).Scan(&repositoryAllowed); err != nil {
		return fmt.Errorf("repository is not authorized by installation")
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) GetReviewChatSubscription(
	ctx context.Context, installationID, chatID, repository string,
) (ReviewChatSubscription, error) {
	return readReviewChatSubscriptionRow(s.db.QueryRowContext(ctx, `SELECT subscription_id,
		installation_id, chat_id, repository, pull_events, review_events, thread_events,
		merge_events, ci_events, notification_mode, enabled, revision,
		created_by_hash, updated_by_hash, created_at, updated_at
		FROM review_chat_subscriptions WHERE installation_id=? AND chat_id=? AND repository=?`,
		installationID, chatID, repository))
}

func readReviewChatSubscriptionTx(ctx context.Context, tx *sql.Tx, installationID, chatID, repository string) (ReviewChatSubscription, error) {
	return readReviewChatSubscriptionRow(tx.QueryRowContext(ctx, `SELECT subscription_id,
		installation_id, chat_id, repository, pull_events, review_events, thread_events,
		merge_events, ci_events, notification_mode, enabled, revision,
		created_by_hash, updated_by_hash, created_at, updated_at
		FROM review_chat_subscriptions WHERE installation_id=? AND chat_id=? AND repository=?`,
		installationID, chatID, repository))
}

func readReviewChatSubscriptionRow(row *sql.Row) (ReviewChatSubscription, error) {
	var subscription ReviewChatSubscription
	var pull, review, thread, merge, ci, enabled int
	var chatID string
	err := row.Scan(&subscription.SubscriptionID, &subscription.InstallationID, &chatID,
		&subscription.Repository, &pull, &review, &thread, &merge, &ci,
		&subscription.NotificationMode, &enabled, &subscription.Revision,
		&subscription.CreatedByHash, &subscription.UpdatedByHash,
		&subscription.CreatedAt, &subscription.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewChatSubscription{}, nil
	}
	if err != nil {
		return ReviewChatSubscription{}, err
	}
	subscription.chatID = chatID
	subscription.ChatIDHash = reviewResourceIdentifierHash(chatID)
	subscription.Enabled = enabled != 0
	subscription.EventGroups = reviewSubscriptionGroups(pull, review, thread, merge, ci)
	return subscription, nil
}

func (s *SQLiteReviewGatewayStore) ListReviewChatSubscriptions(
	ctx context.Context, installationID, chatID, repository string, enabledOnly bool,
) ([]ReviewChatSubscription, error) {
	query := `SELECT subscription_id, installation_id, chat_id, repository,
		pull_events, review_events, thread_events, merge_events, ci_events,
		notification_mode, enabled, revision, created_by_hash, updated_by_hash,
		created_at, updated_at FROM review_chat_subscriptions WHERE 1=1`
	args := []interface{}{}
	for _, filter := range []struct{ column, value string }{{"installation_id", installationID}, {"chat_id", chatID}, {"repository", repository}} {
		if strings.TrimSpace(filter.value) != "" {
			query += " AND " + filter.column + "=?"
			args = append(args, strings.TrimSpace(filter.value))
		}
	}
	if enabledOnly {
		query += " AND enabled=1"
	}
	query += " ORDER BY installation_id, chat_id, repository"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ReviewChatSubscription{}
	for rows.Next() {
		var subscription ReviewChatSubscription
		var pull, review, thread, merge, ci, enabled int
		var chatIDValue string
		if err := rows.Scan(&subscription.SubscriptionID, &subscription.InstallationID, &chatIDValue,
			&subscription.Repository, &pull, &review, &thread, &merge, &ci,
			&subscription.NotificationMode, &enabled, &subscription.Revision,
			&subscription.CreatedByHash, &subscription.UpdatedByHash,
			&subscription.CreatedAt, &subscription.UpdatedAt); err != nil {
			return nil, err
		}
		subscription.chatID = chatIDValue
		subscription.ChatIDHash = reviewResourceIdentifierHash(chatIDValue)
		subscription.Enabled = enabled != 0
		subscription.EventGroups = reviewSubscriptionGroups(pull, review, thread, merge, ci)
		result = append(result, subscription)
	}
	return result, rows.Err()
}

func (s *SQLiteReviewGatewayStore) SetDefaultReviewRepository(
	ctx context.Context, installationID, chatID, repository, actorID string,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	change := ReviewSubscriptionChange{InstallationID: installationID, ChatID: chatID, Repository: repository, ActorID: actorID}
	if err := requireReviewSubscriptionAdministrator(ctx, tx, change); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE chat_repository_bindings SET is_default=0
		WHERE installation_id=? AND chat_id=?`, installationID, chatID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE chat_repository_bindings SET is_default=1
		WHERE installation_id=? AND chat_id=? AND repository=? AND enabled=1`, installationID, chatID, repository)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return fmt.Errorf("default repository target is not an enabled binding")
	}
	return tx.Commit()
}

func reviewSubscriptionGroupInt(groups []string, target string) int {
	if containsReviewGatewayString(groups, target) {
		return 1
	}
	return 0
}

func reviewSubscriptionGroups(pull, review, thread, merge, ci int) []string {
	result := []string{}
	for _, value := range []struct {
		name    string
		enabled int
	}{
		{ReviewSubscriptionGroupPulls, pull}, {ReviewSubscriptionGroupReviews, review},
		{ReviewSubscriptionGroupThreads, thread}, {ReviewSubscriptionGroupMerge, merge},
		{ReviewSubscriptionGroupCI, ci},
	} {
		if value.enabled != 0 {
			result = append(result, value.name)
		}
	}
	return result
}

func equalReviewSubscriptionStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
