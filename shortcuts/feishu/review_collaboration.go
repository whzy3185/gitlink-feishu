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

	"github.com/gitlink-org/gitlink-cli/internal/collab"
)

const (
	reviewCollaborationItemSchema   = "review.collaboration-item/v1"
	reviewCollaborationBundleSchema = "review.collaboration-bundle/v1"
)

type ReviewCollaborationItem struct {
	SchemaVersion       string `json:"schema_version"`
	PRKey               string `json:"pr_key"`
	Repository          string `json:"repository"`
	PRNumber            int    `json:"pr_number"`
	ChatID              string `json:"chat_id,omitempty"`
	ReviewStage         string `json:"review_stage"`
	Decision            string `json:"decision"`
	CollectionStatus    string `json:"collection_status"`
	HeadSHA             string `json:"head_sha,omitempty"`
	SourceFingerprint   string `json:"source_fingerprint,omitempty"`
	AssignedTo          string `json:"assigned_to,omitempty"`
	CollaborationStatus string `json:"collaboration_status"`
	DueAt               string `json:"due_at,omitempty"`
	Archived            bool   `json:"archived"`
	UpdatedBy           string `json:"updated_by,omitempty"`
	UpdatedAt           string `json:"updated_at"`
}

type ReviewCollaborationBundle struct {
	SchemaVersion string                      `json:"schema_version"`
	UniqueKey     string                      `json:"unique_key"`
	Item          ReviewCollaborationItem     `json:"item"`
	BitableRecord BitableRecord               `json:"bitable_record"`
	DocMarkdown   string                      `json:"doc_markdown"`
	Card          Card                        `json:"card"`
	Task          *TaskCandidate              `json:"task,omitempty"`
	SyncTargets   []ReviewCollaborationTarget `json:"sync_targets"`
	GitLinkWrites int                         `json:"gitlink_writes"`
	Canonical     collab.WorkItem             `json:"canonical"`
}

type ReviewCollaborationTarget struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
}

type ReviewCollaborationStore interface {
	ApplyCollaborationAction(context.Context, ReviewGatewayJob, time.Time) (ReviewCollaborationItem, error)
	UpsertCollaborationFacts(context.Context, ReviewGatewayJob, ReviewGatewayExecutionResult, time.Time) (ReviewCollaborationItem, error)
	ListCollaborationItems(context.Context, string, string) ([]ReviewCollaborationItem, error)
}

func (s *SQLiteReviewGatewayStore) ApplyCollaborationAction(
	ctx context.Context,
	job ReviewGatewayJob,
	now time.Time,
) (ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration store is unavailable")
	}
	if job.Repository == "" || job.PRNumber <= 0 {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration target is required")
	}
	now = now.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	defer tx.Rollback()
	item, err := readReviewCollaborationItem(ctx, tx, reviewCollaborationPRKey(job.Repository, job.PRNumber))
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if item.PRKey == "" {
		item = newReviewCollaborationItem(job, now)
	}
	before := item
	switch job.Action {
	case "claim_review":
		if item.AssignedTo != "" && item.AssignedTo != job.RequestedBy {
			return ReviewCollaborationItem{}, fmt.Errorf("PR #%d is already claimed by another reviewer", job.PRNumber)
		}
		item.AssignedTo = job.RequestedBy
		item.CollaborationStatus = "reviewing"
	case "release_review":
		if item.AssignedTo == "" {
			return ReviewCollaborationItem{}, fmt.Errorf("PR #%d is not currently claimed", job.PRNumber)
		}
		if item.AssignedTo != job.RequestedBy {
			return ReviewCollaborationItem{}, fmt.Errorf("only the current reviewer can release PR #%d", job.PRNumber)
		}
		item.AssignedTo = ""
		item.CollaborationStatus = "unassigned"
		item.DueAt = ""
	case "set_review_deadline":
		if item.AssignedTo == "" || item.AssignedTo != job.RequestedBy {
			return ReviewCollaborationItem{}, fmt.Errorf("claim PR #%d before setting its deadline", job.PRNumber)
		}
		due, parseErr := time.Parse("2006-01-02", strings.TrimSpace(job.Argument))
		if parseErr != nil {
			return ReviewCollaborationItem{}, fmt.Errorf("deadline must use YYYY-MM-DD")
		}
		item.DueAt = due.Format("2006-01-02")
	default:
		return ReviewCollaborationItem{}, fmt.Errorf("unsupported collaboration action %q", job.Action)
	}
	item.ChatID = job.ChatID
	item.UpdatedBy = job.RequestedBy
	item.UpdatedAt = now.Format(time.RFC3339Nano)
	if err := writeReviewCollaborationItem(ctx, tx, item); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := writeReviewCollaborationAudit(ctx, tx, before, item, job, now); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewCollaborationItem{}, err
	}
	return item, nil
}

func (s *SQLiteReviewGatewayStore) UpsertCollaborationFacts(
	ctx context.Context,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewCollaborationItem, error) {
	if s == nil || s.db == nil {
		return ReviewCollaborationItem{}, fmt.Errorf("review collaboration store is unavailable")
	}
	now = now.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	defer tx.Rollback()
	key := reviewCollaborationPRKey(job.Repository, job.PRNumber)
	item, err := readReviewCollaborationItem(ctx, tx, key)
	if err != nil {
		return ReviewCollaborationItem{}, err
	}
	if item.PRKey == "" {
		item = newReviewCollaborationItem(job, now)
	}
	if result.CollectionStatus == "complete" && !result.Partial {
		item.ReviewStage = firstNonEmpty(result.ReviewStage, item.ReviewStage)
		item.Decision = firstNonEmpty(result.Decision, item.Decision)
		item.CollectionStatus = result.CollectionStatus
		item.HeadSHA = result.HeadSHA
		item.SourceFingerprint = result.SourceFingerprint
		if result.GitLinkState == "merged" || result.GitLinkState == "closed" {
			item.Archived = true
			item.CollaborationStatus = "archived"
		}
	} else if item.CollectionStatus == "pending" {
		item.CollectionStatus = firstNonEmpty(result.CollectionStatus, "partial")
	}
	item.ChatID = job.ChatID
	item.UpdatedBy = job.RequestedBy
	item.UpdatedAt = now.Format(time.RFC3339Nano)
	if err := writeReviewCollaborationItem(ctx, tx, item); err != nil {
		return ReviewCollaborationItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewCollaborationItem{}, err
	}
	return item, nil
}

func (s *SQLiteReviewGatewayStore) ListCollaborationItems(
	ctx context.Context,
	repository,
	assignedTo string,
) ([]ReviewCollaborationItem, error) {
	query := `SELECT pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
		FROM review_collaboration_items WHERE archived = 0`
	args := []interface{}{}
	if strings.TrimSpace(repository) != "" {
		query += " AND repository = ?"
		args = append(args, strings.TrimSpace(repository))
	}
	if strings.TrimSpace(assignedTo) != "" {
		query += " AND assigned_to = ?"
		args = append(args, strings.TrimSpace(assignedTo))
	}
	query += " ORDER BY due_at = '', due_at, updated_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ReviewCollaborationItem{}
	for rows.Next() {
		item, scanErr := scanReviewCollaborationItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type reviewCollaborationScanner interface {
	Scan(...interface{}) error
}

func readReviewCollaborationItem(ctx context.Context, tx *sql.Tx, key string) (ReviewCollaborationItem, error) {
	row := tx.QueryRowContext(ctx, `SELECT pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
		FROM review_collaboration_items WHERE pr_key = ?`, key)
	item, err := scanReviewCollaborationItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewCollaborationItem{}, nil
	}
	return item, err
}

func scanReviewCollaborationItem(scanner reviewCollaborationScanner) (ReviewCollaborationItem, error) {
	item := ReviewCollaborationItem{SchemaVersion: reviewCollaborationItemSchema}
	var archived int
	err := scanner.Scan(
		&item.PRKey,
		&item.Repository,
		&item.PRNumber,
		&item.ChatID,
		&item.ReviewStage,
		&item.Decision,
		&item.CollectionStatus,
		&item.HeadSHA,
		&item.SourceFingerprint,
		&item.AssignedTo,
		&item.CollaborationStatus,
		&item.DueAt,
		&archived,
		&item.UpdatedBy,
		&item.UpdatedAt,
	)
	item.Archived = archived != 0
	return item, err
}

func writeReviewCollaborationItem(ctx context.Context, tx *sql.Tx, item ReviewCollaborationItem) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO review_collaboration_items (
		pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(pr_key) DO UPDATE SET
		chat_id=excluded.chat_id,
		review_stage=excluded.review_stage,
		decision=excluded.decision,
		collection_status=excluded.collection_status,
		head_sha=excluded.head_sha,
		source_fingerprint=excluded.source_fingerprint,
		assigned_to=excluded.assigned_to,
		collaboration_status=excluded.collaboration_status,
		due_at=excluded.due_at,
		archived=excluded.archived,
		updated_by=excluded.updated_by,
		updated_at=excluded.updated_at`,
		item.PRKey,
		item.Repository,
		item.PRNumber,
		item.ChatID,
		item.ReviewStage,
		item.Decision,
		item.CollectionStatus,
		item.HeadSHA,
		item.SourceFingerprint,
		item.AssignedTo,
		item.CollaborationStatus,
		item.DueAt,
		boolToReviewCollaborationInt(item.Archived),
		item.UpdatedBy,
		item.UpdatedAt,
	)
	return err
}

func writeReviewCollaborationAudit(
	ctx context.Context,
	tx *sql.Tx,
	before,
	after ReviewCollaborationItem,
	job ReviewGatewayJob,
	now time.Time,
) error {
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	auditID := stableKey("review-audit", job.JobID, job.Action)
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO review_collaboration_audit (
		audit_id, pr_key, action, actor_id, before_json, after_json, source_job_id, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		auditID,
		after.PRKey,
		job.Action,
		job.RequestedBy,
		string(beforeJSON),
		string(afterJSON),
		job.JobID,
		now.Format(time.RFC3339Nano),
	)
	return err
}

func newReviewCollaborationItem(job ReviewGatewayJob, now time.Time) ReviewCollaborationItem {
	return ReviewCollaborationItem{
		SchemaVersion:       reviewCollaborationItemSchema,
		PRKey:               reviewCollaborationPRKey(job.Repository, job.PRNumber),
		Repository:          job.Repository,
		PRNumber:            job.PRNumber,
		ChatID:              job.ChatID,
		ReviewStage:         "unreviewed",
		Decision:            "pending",
		CollectionStatus:    "pending",
		CollaborationStatus: "unassigned",
		UpdatedBy:           job.RequestedBy,
		UpdatedAt:           now.UTC().Format(time.RFC3339Nano),
	}
}

func reviewCollaborationPRKey(repository string, number int) string {
	return fmt.Sprintf("%s#%d", strings.TrimSpace(repository), number)
}

func boolToReviewCollaborationInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func BuildReviewCollaborationBundle(item ReviewCollaborationItem) ReviewCollaborationBundle {
	uniqueKey := stableKey("review-work-item", item.Repository, fmt.Sprintf("%d", item.PRNumber))
	fields := map[string]interface{}{
		"pr_key":               item.PRKey,
		"repository":           item.Repository,
		"pr_number":            item.PRNumber,
		"review_stage":         item.ReviewStage,
		"decision":             item.Decision,
		"collection_status":    item.CollectionStatus,
		"head_sha":             item.HeadSHA,
		"source_fingerprint":   item.SourceFingerprint,
		"assigned_to":          item.AssignedTo,
		"collaboration_status": item.CollaborationStatus,
		"due_at":               item.DueAt,
		"archived":             item.Archived,
		"updated_at":           item.UpdatedAt,
	}
	lines := []string{
		fmt.Sprintf("# %s PR #%d Review 协作", item.Repository, item.PRNumber),
		"",
		fmt.Sprintf("- Review 阶段：%s", item.ReviewStage),
		fmt.Sprintf("- GitLink 决策：%s", item.Decision),
		fmt.Sprintf("- 协作状态：%s", item.CollaborationStatus),
		fmt.Sprintf("- 负责人：%s", firstNonEmpty(item.AssignedTo, "未认领")),
		fmt.Sprintf("- 截止时间：%s", firstNonEmpty(item.DueAt, "未设置")),
		fmt.Sprintf("- 数据完整性：%s", item.CollectionStatus),
		"- GitLink 写入：0",
	}
	task := &TaskCandidate{
		UniqueKey:        uniqueKey,
		Title:            fmt.Sprintf("Review %s PR #%d", item.Repository, item.PRNumber),
		Description:      fmt.Sprintf("阶段=%s；决策=%s；由 GitLink 只读事实生成。", item.ReviewStage, item.Decision),
		SourceType:       "gitlink_pr_review",
		SourceKey:        item.PRKey,
		Repository:       item.Repository,
		Priority:         reviewCollaborationPriority(item),
		TaskType:         "pr_review",
		RecommendedOwner: item.AssignedTo,
		Status:           item.CollaborationStatus,
		DueHint:          item.DueAt,
		GitLinkURL:       fmt.Sprintf("https://www.gitlink.org.cn/%s/pulls/%d", item.Repository, item.PRNumber),
	}
	if item.Archived {
		task = nil
	}
	targets := []ReviewCollaborationTarget{
		{Resource: "feishu_card", Action: "reply_or_update", Reason: "notify the source conversation"},
		{Resource: "feishu_bitable", Action: "upsert", Reason: "stable unique key preserves human fields"},
		{Resource: "feishu_doc", Action: "append_snapshot", Reason: "create an auditable review record"},
	}
	if task != nil {
		targets = append(targets, ReviewCollaborationTarget{Resource: "feishu_task", Action: "upsert", Reason: "one task per PR work item"})
	}
	return ReviewCollaborationBundle{
		SchemaVersion: reviewCollaborationBundleSchema,
		UniqueKey:     uniqueKey,
		Item:          item,
		BitableRecord: BitableRecord{UniqueKey: uniqueKey, Fields: fields},
		DocMarkdown:   strings.Join(lines, "\n"),
		Card:          buildReviewCollaborationCard(item),
		Task:          task,
		SyncTargets:   targets,
		GitLinkWrites: 0,
		Canonical: collab.WorkItem{
			SchemaVersion:       collab.WorkItemSchema,
			PRKey:               item.PRKey,
			Repository:          item.Repository,
			PRNumber:            item.PRNumber,
			GitLinkURL:          fmt.Sprintf("https://www.gitlink.org.cn/%s/pulls/%d", item.Repository, item.PRNumber),
			ReviewStage:         item.ReviewStage,
			Decision:            item.Decision,
			CollectionStatus:    item.CollectionStatus,
			HeadSHA:             item.HeadSHA,
			SourceFingerprint:   item.SourceFingerprint,
			AssignedTo:          item.AssignedTo,
			CollaborationStatus: item.CollaborationStatus,
			DueAt:               item.DueAt,
			Archived:            item.Archived,
			NextStep:            reviewCollaborationNextStep(item),
			Evidence: []collab.Evidence{
				{Kind: "gitlink_head", Label: "Head SHA", Value: item.HeadSHA},
				{Kind: "source_fingerprint", Label: "Source fingerprint", Value: item.SourceFingerprint},
			},
			UpdatedAt: item.UpdatedAt,
		},
	}
}

func reviewCollaborationNextStep(item ReviewCollaborationItem) string {
	if item.Archived {
		return "归档协作材料"
	}
	if item.AssignedTo == "" {
		return "由 Owner 团队认领 Review"
	}
	switch item.Decision {
	case "blocked", "rejected", "changes_pending":
		return "联系贡献者处理阻塞项后重新 Review"
	case "approved":
		return "由拥有者执行最终合并判断"
	default:
		return "按证据完成当前 patchset Review"
	}
}

func buildReviewCollaborationCard(item ReviewCollaborationItem) Card {
	elements := []interface{}{
		map[string]interface{}{
			"tag": "markdown",
			"content": fmt.Sprintf(
				"**阶段**：%s\n**决策**：%s\n**协作状态**：%s\n**负责人**：%s\n**截止时间**：%s\n**GitLink 写入**：0",
				item.ReviewStage,
				item.Decision,
				item.CollaborationStatus,
				firstNonEmpty(item.AssignedTo, "未认领"),
				firstNonEmpty(item.DueAt, "未设置"),
			),
		},
	}
	return baseCard(
		fmt.Sprintf("%s PR #%d Review", item.Repository, item.PRNumber),
		"blue",
		elements,
	)
}

func reviewCollaborationPriority(item ReviewCollaborationItem) string {
	switch item.Decision {
	case "blocked", "rejected":
		return "high"
	case "changes_pending", "pending":
		return "medium"
	default:
		return "low"
	}
}

func sortReviewCollaborationItems(items []ReviewCollaborationItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DueAt != items[j].DueAt {
			return items[i].DueAt < items[j].DueAt
		}
		return items[i].PRKey < items[j].PRKey
	})
}
