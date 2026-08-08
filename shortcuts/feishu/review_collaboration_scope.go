package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrReviewCollaborationConflict = errors.New("review collaboration state changed concurrently")

// reviewPRSnapshot contains facts whose authority is GitLink. It is shared by
// chats only inside one installation and never stores Feishu collaboration
// ownership or deadlines.
type reviewPRSnapshot struct {
	SnapshotKey       string
	InstallationID    string
	Repository        string
	PRNumber          int
	ReviewStage       string
	Decision          string
	CollectionStatus  string
	HeadSHA           string
	SourceFingerprint string
	GitLinkState      string
	Archived          bool
	UpdatedAt         string
}

// reviewChatCollaboration contains human-maintained state. Its identity always
// includes installation and chat so the same PR can be coordinated
// independently by multiple owner teams.
type reviewChatCollaboration struct {
	CollaborationKey    string
	InstallationID      string
	ChatID              string
	Repository          string
	PRNumber            int
	AssignedTo          string
	AssignedDisplayName string
	CollaborationStatus string
	DueAt               string
	UpdatedBy           string
	UpdatedAt           string
}

func reviewSnapshotScopeKey(installationID, repository string, number int) string {
	return stableKey(
		"review-snapshot",
		firstNonEmpty(installationID, "legacy"),
		repository,
		fmt.Sprintf("%d", number),
	)
}

func reviewCollaborationScopeKey(installationID, chatID, repository string, number int) string {
	return stableKey(
		"review-collaboration",
		firstNonEmpty(installationID, "legacy"),
		firstNonEmpty(chatID, "unknown-chat"),
		repository,
		fmt.Sprintf("%d", number),
	)
}

func newScopedReviewSnapshot(job ReviewGatewayJob, now time.Time) reviewPRSnapshot {
	return reviewPRSnapshot{
		SnapshotKey:      reviewSnapshotScopeKey(job.InstallationID, job.Repository, job.PRNumber),
		InstallationID:   firstNonEmpty(job.InstallationID, "legacy"),
		Repository:       strings.TrimSpace(job.Repository),
		PRNumber:         job.PRNumber,
		ReviewStage:      "unreviewed",
		Decision:         "pending",
		CollectionStatus: "pending",
		UpdatedAt:        now.UTC().Format(time.RFC3339Nano),
	}
}

func newScopedReviewCollaboration(job ReviewGatewayJob, now time.Time) reviewChatCollaboration {
	return reviewChatCollaboration{
		CollaborationKey:    reviewCollaborationScopeKey(job.InstallationID, job.ChatID, job.Repository, job.PRNumber),
		InstallationID:      firstNonEmpty(job.InstallationID, "legacy"),
		ChatID:              strings.TrimSpace(job.ChatID),
		Repository:          strings.TrimSpace(job.Repository),
		PRNumber:            job.PRNumber,
		CollaborationStatus: "unassigned",
		UpdatedBy:           job.RequestedBy,
		UpdatedAt:           now.UTC().Format(time.RFC3339Nano),
	}
}

func (s *SQLiteReviewGatewayStore) applyScopedCollaborationAction(
	ctx context.Context,
	job ReviewGatewayJob,
	now time.Time,
) (ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration store is unavailable")
	}
	if strings.TrimSpace(job.Repository) == "" || job.PRNumber <= 0 || strings.TrimSpace(job.ChatID) == "" {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration installation, chat, and PR target are required")
	}
	now = now.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	defer tx.Rollback()
	presentation, err := readReviewPRPresentationTx(ctx, tx, job)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if !presentation.Complete() || !presentation.Matches(job) {
		return ReviewCollaborationItem{}, ErrCompleteReviewPRPresentationRequired
	}

	snapshot, err := readScopedReviewSnapshot(ctx, tx, job)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if snapshot.SnapshotKey == "" {
		snapshot = reviewSnapshotFromPresentation(presentation, now)
	}
	state, err := readScopedReviewCollaboration(ctx, tx, job)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if state.CollaborationKey == "" {
		state, err = migrateLegacyReviewCollaborationForJob(ctx, tx, job, now)
		if err != nil {
			return ReviewCollaborationItem{}, err
		}
	}
	statePersisted := state.CollaborationKey != ""
	if state.CollaborationKey == "" {
		state = newScopedReviewCollaboration(job, now)
	}
	stateBeforeMutation := state
	actionOutcome := "updated"
	before := mergeScopedReviewItem(snapshot, state)
	if snapshot.Archived {
		return ReviewCollaborationItem{}, fmt.Errorf("PR #%d is archived and no longer accepts collaboration actions", job.PRNumber)
	}

	switch job.Action {
	case "claim_review":
		if state.AssignedTo != "" && state.AssignedTo != job.RequestedBy {
			return ReviewCollaborationItem{}, fmt.Errorf(
				"PR #%d 当前由%s负责。请先由当前负责人或管理员取消领取",
				job.PRNumber,
				reviewCollaborationAssigneeDisplayName(state),
			)
		}
		alreadyResponsible := state.AssignedTo == job.RequestedBy
		if alreadyResponsible {
			actionOutcome = "already_responsible"
		} else {
			actionOutcome = "claimed"
		}
		state.AssignedTo = job.RequestedBy
		if !alreadyResponsible || strings.TrimSpace(state.AssignedDisplayName) == "" {
			state.AssignedDisplayName = strings.TrimSpace(job.RequestedDisplayName)
		}
		state.CollaborationStatus = "reviewing"
	case "release_review":
		if state.AssignedTo == "" {
			actionOutcome = "no_responsible"
			break
		}
		if state.AssignedTo != job.RequestedBy && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf(
				"当前 PR 由%s负责，只有当前负责人或协作管理员可以取消领取",
				reviewCollaborationAssigneeDisplayName(state),
			)
		}
		state.AssignedTo = ""
		state.AssignedDisplayName = ""
		state.CollaborationStatus = "unassigned"
		actionOutcome = "released"
	case "set_review_deadline":
		if state.AssignedTo == "" && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf("请先领取 PR #%d，再设置审查截止时间", job.PRNumber)
		}
		if state.AssignedTo != job.RequestedBy && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf(
				"当前 PR 由%s负责，只有当前负责人或协作管理员可以设置审查截止时间",
				reviewCollaborationAssigneeDisplayName(state),
			)
		}
		due, parseErr := time.Parse("2006-01-02", strings.TrimSpace(job.Argument))
		if parseErr != nil {
			return ReviewCollaborationItem{}, fmt.Errorf("日期格式应为 YYYY-MM-DD，例如 2026-08-10")
		}
		if due.Format("2006-01-02") < now.Format("2006-01-02") {
			return ReviewCollaborationItem{}, fmt.Errorf("审查截止时间不能早于今天")
		}
		if state.DueAt == "" {
			actionOutcome = "deadline_set"
		} else if state.DueAt == due.Format("2006-01-02") {
			actionOutcome = "deadline_unchanged"
		} else {
			actionOutcome = "deadline_updated"
		}
		state.DueAt = due.Format("2006-01-02")
	case "clear_review_deadline":
		if state.AssignedTo == "" && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf("请先领取 PR #%d，再清除审查截止时间", job.PRNumber)
		}
		if state.AssignedTo != "" && state.AssignedTo != job.RequestedBy && !job.CollaborationAdmin {
			return ReviewCollaborationItem{}, fmt.Errorf(
				"当前 PR 由%s负责，只有当前负责人或协作管理员可以清除审查截止时间",
				reviewCollaborationAssigneeDisplayName(state),
			)
		}
		if state.DueAt == "" {
			actionOutcome = "deadline_absent"
		} else {
			actionOutcome = "deadline_cleared"
		}
		state.DueAt = ""
	default:
		return ReviewCollaborationItem{}, fmt.Errorf("unsupported collaboration action %q", job.Action)
	}
	state.UpdatedBy = job.RequestedBy
	state.UpdatedAt = now.Format(time.RFC3339Nano)
	if err := writeScopedReviewCollaborationAction(ctx, tx, stateBeforeMutation, state, statePersisted); err != nil {
		return ReviewCollaborationItem{}, err
	}
	after := mergeScopedReviewItem(snapshot, state)
	after.ActionOutcome = actionOutcome
	if err := writeReviewCollaborationAudit(ctx, tx, before, after, job, now); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewCollaborationItem{}, err
	}
	return after, nil
}

func reviewCollaborationAssigneeDisplayName(state reviewChatCollaboration) string {
	if value := strings.TrimSpace(state.AssignedDisplayName); value != "" {
		return truncateReviewGatewayText(value, 80)
	}
	return "当前负责人"
}

func writeScopedReviewCollaborationAction(
	ctx context.Context,
	tx *sql.Tx,
	before,
	after reviewChatCollaboration,
	persisted bool,
) error {
	var (
		result sql.Result
		err    error
	)
	if !persisted {
		result, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO review_collaboration_states (
			collaboration_key, installation_id, chat_id, repository, pr_number,
			assigned_to, assigned_display_name, collaboration_status, due_at, updated_by, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			after.CollaborationKey, after.InstallationID, after.ChatID, after.Repository, after.PRNumber,
			after.AssignedTo, after.AssignedDisplayName, after.CollaborationStatus,
			after.DueAt, after.UpdatedBy, after.UpdatedAt,
		)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE review_collaboration_states SET
			assigned_to=?, assigned_display_name=?, collaboration_status=?, due_at=?,
			updated_by=?, updated_at=?
			WHERE collaboration_key=? AND assigned_to=? AND assigned_display_name=?
			  AND collaboration_status=? AND due_at=? AND updated_at=?`,
			after.AssignedTo, after.AssignedDisplayName, after.CollaborationStatus,
			after.DueAt, after.UpdatedBy, after.UpdatedAt, before.CollaborationKey,
			before.AssignedTo, before.AssignedDisplayName, before.CollaborationStatus,
			before.DueAt, before.UpdatedAt,
		)
	}
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrReviewCollaborationConflict
	}
	return nil
}

func (s *SQLiteReviewGatewayStore) upsertScopedCollaborationFacts(
	ctx context.Context,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration store is unavailable")
	}
	if strings.TrimSpace(job.Repository) == "" || job.PRNumber <= 0 || strings.TrimSpace(job.ChatID) == "" {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration installation, chat, and PR target are required")
	}
	now = now.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	defer tx.Rollback()
	if _, err := upsertReviewPRPresentationTx(ctx, tx, job, result, now); err != nil {
		return ReviewCollaborationItem{}, err
	}

	snapshot, err := readScopedReviewSnapshot(ctx, tx, job)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if snapshot.SnapshotKey == "" {
		snapshot = newScopedReviewSnapshot(job, now)
	}
	if result.CollectionStatus == "complete" && !result.Partial {
		snapshot.ReviewStage = firstNonEmpty(result.ReviewStage, snapshot.ReviewStage)
		snapshot.Decision = firstNonEmpty(result.Decision, snapshot.Decision)
		snapshot.CollectionStatus = result.CollectionStatus
		snapshot.HeadSHA = result.HeadSHA
		snapshot.SourceFingerprint = result.SourceFingerprint
		snapshot.GitLinkState = result.GitLinkState
		if result.GitLinkState == "merged" || result.GitLinkState == "closed" {
			snapshot.Archived = true
		}
	} else if snapshot.CollectionStatus == "pending" {
		snapshot.CollectionStatus = firstNonEmpty(result.CollectionStatus, "partial")
	}
	snapshot.UpdatedAt = now.Format(time.RFC3339Nano)
	if err := writeScopedReviewSnapshot(ctx, tx, snapshot); err != nil {
		return ReviewCollaborationItem{}, err
	}

	state, err := readScopedReviewCollaboration(ctx, tx, job)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if state.CollaborationKey == "" {
		state, err = migrateLegacyReviewCollaborationForJob(ctx, tx, job, now)
		if err != nil {
			return ReviewCollaborationItem{}, err
		}
	}
	if state.CollaborationKey == "" {
		state = newScopedReviewCollaboration(job, now)
	}
	if snapshot.Archived {
		state.CollaborationStatus = "archived"
	}
	state.UpdatedBy = job.RequestedBy
	state.UpdatedAt = now.Format(time.RFC3339Nano)
	if err := writeScopedReviewCollaboration(ctx, tx, state); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewCollaborationItem{}, err
	}
	return mergeScopedReviewItem(snapshot, state), nil
}

func reviewSnapshotFromPresentation(presentation ReviewPRPresentation, now time.Time) reviewPRSnapshot {
	return reviewPRSnapshot{
		SnapshotKey:       reviewSnapshotScopeKey(presentation.InstallationID, presentation.Repository, presentation.PRNumber),
		InstallationID:    presentation.InstallationID,
		Repository:        presentation.Repository,
		PRNumber:          presentation.PRNumber,
		ReviewStage:       firstNonEmpty(presentation.ReviewStage, "unreviewed"),
		Decision:          firstNonEmpty(presentation.Decision, "pending"),
		CollectionStatus:  presentation.CollectionStatus,
		HeadSHA:           presentation.HeadSHA,
		SourceFingerprint: presentation.SourceFingerprint,
		GitLinkState:      presentation.GitLinkState,
		Archived:          presentation.GitLinkState == "merged" || presentation.GitLinkState == "closed",
		UpdatedAt:         now.UTC().Format(time.RFC3339Nano),
	}
}

func (s *SQLiteReviewGatewayStore) listScopedCollaborationItems(
	ctx context.Context,
	installationID,
	chatID,
	repository,
	assignedTo string,
) ([]ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("review collaboration store is unavailable")
	}
	query := `SELECT
		p.snapshot_key, p.installation_id, p.repository, p.pr_number,
		p.review_stage, p.decision, p.collection_status, p.head_sha,
		p.source_fingerprint, p.gitlink_state, p.archived, p.updated_at,
		c.collaboration_key, c.chat_id, c.assigned_to, c.assigned_display_name,
		c.collaboration_status, c.due_at, c.updated_by, c.updated_at
		FROM review_collaboration_states c
		JOIN review_pr_snapshots p
		  ON p.installation_id = c.installation_id
		 AND p.repository = c.repository
		 AND p.pr_number = c.pr_number
		WHERE c.installation_id = ? AND c.chat_id = ? AND p.archived = 0`
	args := []interface{}{firstNonEmpty(installationID, "legacy"), strings.TrimSpace(chatID)}
	if strings.TrimSpace(repository) != "" {
		query += " AND c.repository = ?"
		args = append(args, strings.TrimSpace(repository))
	}
	if strings.TrimSpace(assignedTo) != "" {
		query += " AND c.assigned_to = ?"
		args = append(args, strings.TrimSpace(assignedTo))
	}
	query += " ORDER BY c.due_at = '', c.due_at, c.updated_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ReviewCollaborationItem{}
	for rows.Next() {
		var snapshot reviewPRSnapshot
		var state reviewChatCollaboration
		var archived int
		if err := rows.Scan(
			&snapshot.SnapshotKey, &snapshot.InstallationID, &snapshot.Repository, &snapshot.PRNumber,
			&snapshot.ReviewStage, &snapshot.Decision, &snapshot.CollectionStatus, &snapshot.HeadSHA,
			&snapshot.SourceFingerprint, &snapshot.GitLinkState, &archived, &snapshot.UpdatedAt,
			&state.CollaborationKey, &state.ChatID, &state.AssignedTo, &state.AssignedDisplayName,
			&state.CollaborationStatus, &state.DueAt, &state.UpdatedBy, &state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		snapshot.Archived = archived != 0
		state.InstallationID = snapshot.InstallationID
		state.Repository = snapshot.Repository
		state.PRNumber = snapshot.PRNumber
		items = append(items, mergeScopedReviewItem(snapshot, state))
	}
	return items, rows.Err()
}

func readScopedReviewSnapshot(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob) (reviewPRSnapshot, error) {
	var snapshot reviewPRSnapshot
	var archived int
	err := tx.QueryRowContext(ctx, `SELECT snapshot_key, installation_id, repository, pr_number,
		review_stage, decision, collection_status, head_sha, source_fingerprint,
		gitlink_state, archived, updated_at
		FROM review_pr_snapshots WHERE snapshot_key = ?`,
		reviewSnapshotScopeKey(job.InstallationID, job.Repository, job.PRNumber),
	).Scan(
		&snapshot.SnapshotKey, &snapshot.InstallationID, &snapshot.Repository, &snapshot.PRNumber,
		&snapshot.ReviewStage, &snapshot.Decision, &snapshot.CollectionStatus, &snapshot.HeadSHA,
		&snapshot.SourceFingerprint, &snapshot.GitLinkState, &archived, &snapshot.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewPRSnapshot{}, nil
	}
	snapshot.Archived = archived != 0
	return snapshot, err
}

func writeScopedReviewSnapshot(ctx context.Context, tx *sql.Tx, snapshot reviewPRSnapshot) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_pr_snapshots (
		snapshot_key, installation_id, repository, pr_number, review_stage, decision,
		collection_status, head_sha, source_fingerprint, gitlink_state, archived, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(snapshot_key) DO UPDATE SET
		review_stage=excluded.review_stage,
		decision=excluded.decision,
		collection_status=excluded.collection_status,
		head_sha=excluded.head_sha,
		source_fingerprint=excluded.source_fingerprint,
		gitlink_state=excluded.gitlink_state,
		archived=excluded.archived,
		updated_at=excluded.updated_at`,
		snapshot.SnapshotKey, snapshot.InstallationID, snapshot.Repository, snapshot.PRNumber,
		snapshot.ReviewStage, snapshot.Decision, snapshot.CollectionStatus, snapshot.HeadSHA,
		snapshot.SourceFingerprint, snapshot.GitLinkState,
		boolToReviewCollaborationInt(snapshot.Archived), snapshot.UpdatedAt,
	)
	return err
}

func readScopedReviewCollaboration(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob) (reviewChatCollaboration, error) {
	var state reviewChatCollaboration
	err := tx.QueryRowContext(ctx, `SELECT collaboration_key, installation_id, chat_id,
		repository, pr_number, assigned_to, assigned_display_name, collaboration_status,
		due_at, updated_by, updated_at
		FROM review_collaboration_states WHERE collaboration_key = ?`,
		reviewCollaborationScopeKey(job.InstallationID, job.ChatID, job.Repository, job.PRNumber),
	).Scan(
		&state.CollaborationKey, &state.InstallationID, &state.ChatID, &state.Repository,
		&state.PRNumber, &state.AssignedTo, &state.AssignedDisplayName,
		&state.CollaborationStatus, &state.DueAt, &state.UpdatedBy, &state.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return reviewChatCollaboration{}, nil
	}
	return state, err
}

func writeScopedReviewCollaboration(ctx context.Context, tx *sql.Tx, state reviewChatCollaboration) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_collaboration_states (
		collaboration_key, installation_id, chat_id, repository, pr_number,
		assigned_to, assigned_display_name, collaboration_status, due_at, updated_by, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(collaboration_key) DO UPDATE SET
		assigned_to=excluded.assigned_to,
		assigned_display_name=excluded.assigned_display_name,
		collaboration_status=excluded.collaboration_status,
		due_at=excluded.due_at,
		updated_by=excluded.updated_by,
		updated_at=excluded.updated_at`,
		state.CollaborationKey, state.InstallationID, state.ChatID, state.Repository, state.PRNumber,
		state.AssignedTo, state.AssignedDisplayName, state.CollaborationStatus,
		state.DueAt, state.UpdatedBy, state.UpdatedAt,
	)
	return err
}

func mergeScopedReviewItem(snapshot reviewPRSnapshot, state reviewChatCollaboration) ReviewCollaborationItem {
	return ReviewCollaborationItem{
		SchemaVersion:       reviewCollaborationItemSchema,
		PRKey:               state.CollaborationKey,
		InstallationID:      snapshot.InstallationID,
		Repository:          snapshot.Repository,
		PRNumber:            snapshot.PRNumber,
		ChatID:              state.ChatID,
		ReviewStage:         snapshot.ReviewStage,
		Decision:            snapshot.Decision,
		CollectionStatus:    snapshot.CollectionStatus,
		HeadSHA:             snapshot.HeadSHA,
		SourceFingerprint:   snapshot.SourceFingerprint,
		AssignedTo:          state.AssignedTo,
		AssignedDisplayName: state.AssignedDisplayName,
		CollaborationStatus: state.CollaborationStatus,
		DueAt:               state.DueAt,
		Archived:            snapshot.Archived,
		UpdatedBy:           state.UpdatedBy,
		UpdatedAt:           firstNonEmpty(state.UpdatedAt, snapshot.UpdatedAt),
	}
}

func migrateLegacyReviewCollaborationForJob(
	ctx context.Context,
	tx *sql.Tx,
	job ReviewGatewayJob,
	now time.Time,
) (reviewChatCollaboration, error) {
	legacy, err := readReviewCollaborationItem(ctx, tx, reviewCollaborationPRKey(job.Repository, job.PRNumber))
	if err != nil || legacy.PRKey == "" || strings.TrimSpace(legacy.ChatID) != strings.TrimSpace(job.ChatID) {
		return reviewChatCollaboration{}, err
	}
	installationRows, err := tx.QueryContext(ctx, `SELECT DISTINCT installation_id
		FROM chat_repository_bindings
		WHERE chat_id = ? AND repository = ? AND enabled = 1
		ORDER BY installation_id`, job.ChatID, job.Repository)
	if err != nil {
		return reviewChatCollaboration{}, fmt.Errorf("resolve legacy Review installation: %w", err)
	}
	installations := []string{}
	for installationRows.Next() {
		var installationID string
		if err := installationRows.Scan(&installationID); err != nil {
			_ = installationRows.Close()
			return reviewChatCollaboration{}, err
		}
		installations = append(installations, installationID)
	}
	if err := installationRows.Close(); err != nil {
		return reviewChatCollaboration{}, err
	}
	if len(installations) != 1 || installations[0] != firstNonEmpty(job.InstallationID, "legacy") {
		// An ambiguous legacy row has no trustworthy installation identity. Keep
		// it in the legacy table instead of leaking its assignee into whichever
		// installation happens to issue the first request after an upgrade.
		return reviewChatCollaboration{}, nil
	}
	state := newScopedReviewCollaboration(job, now)
	state.AssignedTo = legacy.AssignedTo
	state.CollaborationStatus = legacy.CollaborationStatus
	state.DueAt = legacy.DueAt
	state.UpdatedBy = legacy.UpdatedBy
	state.UpdatedAt = firstNonEmpty(legacy.UpdatedAt, state.UpdatedAt)
	if err := writeScopedReviewCollaboration(ctx, tx, state); err != nil {
		return reviewChatCollaboration{}, err
	}
	if err := DiscoverLegacyReviewResourceMigrations(ctx, tx, now); err != nil {
		return reviewChatCollaboration{}, err
	}
	return state, nil
}

func migrateLegacyReviewResources(
	ctx context.Context,
	tx *sql.Tx,
	repository string,
	prNumber int,
	collaborationKey string,
) error {
	_ = repository
	_ = prNumber
	_ = collaborationKey
	return DiscoverLegacyReviewResourceMigrations(ctx, tx, time.Now().UTC())
}

// migrateLegacyReviewCollaborationScope is intentionally idempotent. It only
// migrates rows that can be mapped to exactly one enabled installation; rows
// without an unambiguous mapping are retained for lazy migration on the next
// scoped request.
func migrateLegacyReviewCollaborationScope(db *sql.DB) error {
	rows, err := db.Query(`SELECT pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
		FROM review_collaboration_items`)
	if err != nil {
		return fmt.Errorf("read legacy Review collaboration rows: %w", err)
	}
	legacyItems := []ReviewCollaborationItem{}
	for rows.Next() {
		item, scanErr := scanReviewCollaborationItem(rows)
		if scanErr != nil {
			_ = rows.Close()
			return fmt.Errorf("scan legacy Review collaboration row: %w", scanErr)
		}
		legacyItems = append(legacyItems, item)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close legacy Review collaboration rows: %w", err)
	}
	for _, legacy := range legacyItems {
		installationRows, queryErr := db.Query(`SELECT installation_id
			FROM chat_repository_bindings
			WHERE chat_id = ? AND repository = ? AND enabled = 1
			ORDER BY installation_id`, legacy.ChatID, legacy.Repository)
		if queryErr != nil {
			return fmt.Errorf("resolve legacy Review installation: %w", queryErr)
		}
		installations := []string{}
		for installationRows.Next() {
			var installationID string
			if err := installationRows.Scan(&installationID); err != nil {
				_ = installationRows.Close()
				return err
			}
			installations = append(installations, installationID)
		}
		_ = installationRows.Close()
		if len(installations) != 1 {
			continue
		}
		job := ReviewGatewayJob{
			InstallationID: installations[0], ChatID: legacy.ChatID,
			Repository: legacy.Repository, PRNumber: legacy.PRNumber,
			RequestedBy: legacy.UpdatedBy,
		}
		snapshot := newScopedReviewSnapshot(job, time.Now().UTC())
		snapshot.ReviewStage = legacy.ReviewStage
		snapshot.Decision = legacy.Decision
		snapshot.CollectionStatus = legacy.CollectionStatus
		snapshot.HeadSHA = legacy.HeadSHA
		snapshot.SourceFingerprint = legacy.SourceFingerprint
		snapshot.Archived = legacy.Archived
		snapshot.UpdatedAt = legacy.UpdatedAt
		state := newScopedReviewCollaboration(job, time.Now().UTC())
		state.AssignedTo = legacy.AssignedTo
		state.CollaborationStatus = legacy.CollaborationStatus
		state.DueAt = legacy.DueAt
		state.UpdatedAt = legacy.UpdatedAt

		tx, beginErr := db.Begin()
		if beginErr != nil {
			return beginErr
		}
		if err := writeScopedReviewSnapshot(context.Background(), tx, snapshot); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate legacy Review snapshot: %w", err)
		}
		if err := writeScopedReviewCollaboration(context.Background(), tx, state); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate legacy Review collaboration: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := DiscoverLegacyReviewResourceMigrations(context.Background(), tx, time.Now().UTC()); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("discover legacy Review resource migrations: %w", err)
	}
	return tx.Commit()
}
