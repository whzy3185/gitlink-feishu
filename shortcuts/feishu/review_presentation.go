package feishu

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const reviewPRPresentationSchema = "review.pr-presentation/v1"

var (
	ErrReviewPRPresentationNotFound         = errors.New("review PR presentation not found")
	ErrCompleteReviewPRPresentationRequired = errors.New("complete PR presentation is required before collaboration action")
	ErrStaleReviewPRPresentation            = errors.New("stale review PR presentation")
)

// ReviewPRPresentation is the bounded, installation-scoped GitLink fact set
// used to rebuild a complete card after a chat-local collaboration action.
// It deliberately excludes chat/user identifiers, credentials, raw payloads,
// file bodies, and full Review thread content.
type ReviewPRPresentation struct {
	PresentationKey     string                      `json:"presentation_key"`
	InstallationID      string                      `json:"installation_id"`
	Repository          string                      `json:"repository"`
	PRNumber            int                         `json:"pr_number"`
	SchemaVersion       string                      `json:"schema_version"`
	Title               string                      `json:"title,omitempty"`
	Author              string                      `json:"author,omitempty"`
	BaseBranch          string                      `json:"base_branch,omitempty"`
	HeadBranch          string                      `json:"head_branch,omitempty"`
	HeadSHA             string                      `json:"head_sha,omitempty"`
	PatchsetID          string                      `json:"patchset_id,omitempty"`
	GitLinkState        string                      `json:"gitlink_state,omitempty"`
	FilesCount          int                         `json:"files_count"`
	CommitsCount        int                         `json:"commits_count"`
	Additions           int                         `json:"additions"`
	Deletions           int                         `json:"deletions"`
	ReviewStage         string                      `json:"review_stage,omitempty"`
	Decision            string                      `json:"decision,omitempty"`
	ReviewCount         int                         `json:"review_count"`
	ThreadCount         int                         `json:"thread_count"`
	OpenThreadCount     int                         `json:"open_thread_count"`
	RiskLevel           string                      `json:"risk_level,omitempty"`
	CollectionStatus    string                      `json:"collection_status"`
	Partial             bool                        `json:"partial"`
	Reviewers           []ReviewGatewayReviewerView `json:"reviewers,omitempty"`
	Unknowns            []string                    `json:"unknowns,omitempty"`
	RecommendedNextStep string                      `json:"recommended_next_step,omitempty"`
	GitLinkURL          string                      `json:"gitlink_url,omitempty"`
	SourceFingerprint   string                      `json:"source_fingerprint,omitempty"`
	ContentFingerprint  string                      `json:"content_fingerprint"`
	SourceCompletedAt   string                      `json:"source_completed_at"`
	UpdatedAt           string                      `json:"updated_at"`
}

func reviewPRPresentationKey(installationID, repository string, number int) string {
	return stableKey("review-pr-presentation", firstNonEmpty(installationID, "legacy"), repository, fmt.Sprintf("%d", number))
}

func (p ReviewPRPresentation) Complete() bool {
	return p.PresentationKey != "" && p.SchemaVersion == reviewPRPresentationSchema &&
		p.CollectionStatus == "complete" && !p.Partial && p.Repository != "" && p.PRNumber > 0
}

func (p ReviewPRPresentation) Matches(job ReviewGatewayJob) bool {
	return p.InstallationID == firstNonEmpty(strings.TrimSpace(job.InstallationID), "legacy") &&
		p.Repository == strings.TrimSpace(job.Repository) && p.PRNumber == job.PRNumber
}

func reviewPRPresentationFromResult(
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewPRPresentation, error) {
	if strings.TrimSpace(job.InstallationID) == "" {
		return ReviewPRPresentation{}, fmt.Errorf("Review installation is required")
	}
	if _, _, err := splitReviewGatewayRepository(job.Repository); err != nil {
		return ReviewPRPresentation{}, err
	}
	if strings.TrimSpace(result.Repository) != strings.TrimSpace(job.Repository) {
		return ReviewPRPresentation{}, fmt.Errorf("Review result repository does not match job scope")
	}
	if result.PRNumber != job.PRNumber || job.PRNumber <= 0 {
		return ReviewPRPresentation{}, fmt.Errorf("Review result PR number does not match job scope")
	}
	sourceCompletedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(result.CompletedAt))
	if err != nil {
		return ReviewPRPresentation{}, fmt.Errorf("Review result completed_at is invalid: %w", err)
	}
	complete := result.CollectionStatus == "complete" && !result.Partial
	if complete && result.PullRequest == nil {
		return ReviewPRPresentation{}, fmt.Errorf("complete Review result requires pull request presentation")
	}
	view := result.PullRequest
	if view == nil {
		view = &ReviewGatewayPullRequestView{}
	}
	reviewers := append([]ReviewGatewayReviewerView(nil), view.Reviewers...)
	for index := range reviewers {
		reviewers[index].Reviewer = truncateReviewPresentationText(reviewers[index].Reviewer, 100)
		reviewers[index].Decision = truncateReviewPresentationText(reviewers[index].Decision, 64)
		reviewers[index].ReviewedAt = truncateReviewPresentationText(reviewers[index].ReviewedAt, 64)
		reviewers[index].Freshness = truncateReviewPresentationText(reviewers[index].Freshness, 32)
	}
	reviewers = deduplicateReviewPresentationReviewers(reviewers)
	sortReviewGatewayReviewers(reviewers)
	unknowns := append([]string(nil), view.Unknowns...)
	for index := range unknowns {
		unknowns[index] = truncateReviewPresentationText(redactReviewGatewayError(unknowns[index]), 300)
	}
	sort.Strings(unknowns)
	unknowns = deduplicateReviewPresentationStrings(unknowns)
	if len(unknowns) > 20 {
		unknowns = unknowns[:20]
	}
	gitLinkURL, err := normalizeReviewPresentationURL(view.GitLinkURL)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	presentation := ReviewPRPresentation{
		PresentationKey:     reviewPRPresentationKey(job.InstallationID, job.Repository, job.PRNumber),
		InstallationID:      firstNonEmpty(strings.TrimSpace(job.InstallationID), "legacy"),
		Repository:          strings.TrimSpace(job.Repository),
		PRNumber:            job.PRNumber,
		SchemaVersion:       reviewPRPresentationSchema,
		Title:               truncateReviewPresentationText(view.Title, 200),
		Author:              truncateReviewPresentationText(view.Author, 100),
		BaseBranch:          truncateReviewPresentationText(view.BaseBranch, 200),
		HeadBranch:          truncateReviewPresentationText(view.HeadBranch, 200),
		HeadSHA:             truncateReviewPresentationText(result.HeadSHA, 64),
		PatchsetID:          truncateReviewPresentationText(view.PatchsetID, 100),
		GitLinkState:        truncateReviewPresentationText(result.GitLinkState, 32),
		FilesCount:          nonNegativeReviewPresentationCount(view.FilesCount),
		CommitsCount:        nonNegativeReviewPresentationCount(view.CommitsCount),
		Additions:           nonNegativeReviewPresentationCount(view.Additions),
		Deletions:           nonNegativeReviewPresentationCount(view.Deletions),
		ReviewStage:         truncateReviewPresentationText(result.ReviewStage, 64),
		Decision:            truncateReviewPresentationText(result.Decision, 64),
		ReviewCount:         nonNegativeReviewPresentationCount(result.ReviewCount),
		ThreadCount:         nonNegativeReviewPresentationCount(result.ThreadCount),
		OpenThreadCount:     nonNegativeReviewPresentationCount(result.OpenThreadCount),
		RiskLevel:           truncateReviewPresentationText(view.RiskLevel, 32),
		CollectionStatus:    truncateReviewPresentationText(firstNonEmpty(result.CollectionStatus, "partial"), 32),
		Partial:             result.Partial || result.CollectionStatus != "complete",
		Reviewers:           reviewers,
		Unknowns:            unknowns,
		RecommendedNextStep: truncateReviewPresentationText(view.RecommendedNextStep, 500),
		GitLinkURL:          gitLinkURL,
		SourceFingerprint:   truncateReviewPresentationText(result.SourceFingerprint, 200),
		SourceCompletedAt:   sourceCompletedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:           now.UTC().Format(time.RFC3339Nano),
	}
	presentation.ContentFingerprint = reviewPRPresentationFingerprint(presentation)
	return presentation, nil
}

func deduplicateReviewPresentationReviewers(values []ReviewGatewayReviewerView) []ReviewGatewayReviewerView {
	result := make([]ReviewGatewayReviewerView, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value.Reviewer))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func deduplicateReviewPresentationStrings(values []string) []string {
	result := make([]string, 0, len(values))
	last := ""
	for _, value := range values {
		if value == "" || value == last {
			continue
		}
		last = value
		result = append(result, value)
	}
	return result
}

func normalizeReviewPresentationURL(value string) (string, error) {
	value = truncateReviewPresentationText(value, 500)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return "", fmt.Errorf("Review presentation GitLink URL is invalid")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "www.gitlink.org.cn" && host != "gitlink.org.cn" {
		return "", fmt.Errorf("Review presentation URL host is not allowed")
	}
	return value, nil
}

func truncateReviewPresentationText(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if max > 0 && len(runes) > max {
		runes = runes[:max]
	}
	return string(runes)
}

func nonNegativeReviewPresentationCount(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func reviewPRPresentationFingerprint(presentation ReviewPRPresentation) string {
	presentation.PresentationKey = ""
	presentation.ContentFingerprint = ""
	presentation.SourceCompletedAt = ""
	presentation.UpdatedAt = ""
	encoded, _ := json.Marshal(presentation)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (p ReviewPRPresentation) Apply(result *ReviewGatewayExecutionResult) {
	if result == nil {
		return
	}
	result.Repository = p.Repository
	result.PRNumber = p.PRNumber
	result.CollectionStatus = p.CollectionStatus
	result.Partial = p.Partial
	result.HeadSHA = p.HeadSHA
	result.SourceFingerprint = p.SourceFingerprint
	result.ReviewStage = p.ReviewStage
	result.Decision = p.Decision
	result.GitLinkState = p.GitLinkState
	result.ReviewCount = p.ReviewCount
	result.ThreadCount = p.ThreadCount
	result.OpenThreadCount = p.OpenThreadCount
	result.CompletedAt = p.SourceCompletedAt
	result.PullRequest = &ReviewGatewayPullRequestView{
		Title:               p.Title,
		Author:              p.Author,
		BaseBranch:          p.BaseBranch,
		HeadBranch:          p.HeadBranch,
		GitLinkURL:          p.GitLinkURL,
		PatchsetID:          p.PatchsetID,
		FilesCount:          p.FilesCount,
		CommitsCount:        p.CommitsCount,
		Additions:           p.Additions,
		Deletions:           p.Deletions,
		RiskLevel:           p.RiskLevel,
		RecommendedNextStep: p.RecommendedNextStep,
		Unknowns:            append([]string(nil), p.Unknowns...),
		Reviewers:           append([]ReviewGatewayReviewerView(nil), p.Reviewers...),
	}
}

func reviewGatewayResultFromPresentation(job ReviewGatewayJob, presentation ReviewPRPresentation) (ReviewGatewayExecutionResult, error) {
	if !presentation.Complete() {
		return ReviewGatewayExecutionResult{}, ErrCompleteReviewPRPresentationRequired
	}
	if !presentation.Matches(job) {
		return ReviewGatewayExecutionResult{}, fmt.Errorf("Review presentation scope does not match job")
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:   reviewGatewayResultSchema,
		JobID:           job.JobID,
		Status:          "completed",
		Mode:            firstNonEmpty(job.Mode, "preview"),
		Action:          job.Action,
		RequestedBy:     job.RequestedBy,
		ReadOnlyGitLink: true,
		MutatesGitLink:  false,
	}
	presentation.Apply(&result)
	return result, nil
}

func (s *SQLiteReviewGatewayStore) SaveReviewPRPresentation(
	ctx context.Context,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewPRPresentation, error) {
	if s == nil || s.db == nil {
		return ReviewPRPresentation{}, fmt.Errorf("review PR presentation store is unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	defer tx.Rollback()
	presentation, err := upsertReviewPRPresentationTx(ctx, tx, job, result, now)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewPRPresentation{}, err
	}
	return presentation, nil
}

func upsertReviewPRPresentationTx(
	ctx context.Context,
	tx *sql.Tx,
	job ReviewGatewayJob,
	result ReviewGatewayExecutionResult,
	now time.Time,
) (ReviewPRPresentation, error) {
	existing, err := readReviewPRPresentationTx(ctx, tx, job)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	if result.Status == "failed" {
		return existing, nil
	}
	incoming, err := reviewPRPresentationFromResult(job, result, now)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	incomingComplete := result.CollectionStatus == "complete" && !result.Partial && result.PullRequest != nil
	if incomingComplete {
		incoming.CollectionStatus = "complete"
		incoming.Partial = false
		incoming.ContentFingerprint = reviewPRPresentationFingerprint(incoming)
	} else if existing.Complete() {
		return existing, nil
	}
	if existing.PresentationKey != "" {
		if existing.ContentFingerprint == incoming.ContentFingerprint {
			return existing, nil
		}
		existingTime, existingErr := time.Parse(time.RFC3339Nano, existing.SourceCompletedAt)
		incomingTime, incomingErr := time.Parse(time.RFC3339Nano, incoming.SourceCompletedAt)
		if existingErr != nil || incomingErr != nil {
			return ReviewPRPresentation{}, fmt.Errorf("compare Review presentation freshness: %w", ErrStaleReviewPRPresentation)
		}
		if incomingTime.Before(existingTime) {
			return ReviewPRPresentation{}, ErrStaleReviewPRPresentation
		}
	}
	if err := writeReviewPRPresentationTx(ctx, tx, incoming); err != nil {
		return ReviewPRPresentation{}, err
	}
	return incoming, nil
}

func (s *SQLiteReviewGatewayStore) GetReviewPRPresentation(ctx context.Context, job ReviewGatewayJob) (ReviewPRPresentation, error) {
	if s == nil || s.db == nil {
		return ReviewPRPresentation{}, fmt.Errorf("review PR presentation store is unavailable")
	}
	row := s.db.QueryRowContext(ctx, reviewPRPresentationSelect+" WHERE presentation_key = ?", reviewPRPresentationKey(job.InstallationID, job.Repository, job.PRNumber))
	presentation, err := scanReviewPRPresentation(row)
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	if presentation.PresentationKey == "" {
		return ReviewPRPresentation{}, ErrReviewPRPresentationNotFound
	}
	return presentation, nil
}

const reviewPRPresentationSelect = `SELECT presentation_key, installation_id, repository, pr_number,
	schema_version, title, author, base_branch, head_branch, head_sha, patchset_id,
	gitlink_state, files_count, commits_count, additions, deletions, review_stage,
	decision, review_count, thread_count, open_thread_count, risk_level,
	collection_status, partial, reviewers_json, unknowns_json, recommended_next_step,
	gitlink_url, source_fingerprint, content_fingerprint, source_completed_at, updated_at
	FROM review_pr_presentations`

func readReviewPRPresentationTx(ctx context.Context, tx *sql.Tx, job ReviewGatewayJob) (ReviewPRPresentation, error) {
	row := tx.QueryRowContext(ctx, reviewPRPresentationSelect+" WHERE presentation_key = ?", reviewPRPresentationKey(job.InstallationID, job.Repository, job.PRNumber))
	return scanReviewPRPresentation(row)
}

type reviewPresentationScanner interface {
	Scan(...interface{}) error
}

func scanReviewPRPresentation(scanner reviewPresentationScanner) (ReviewPRPresentation, error) {
	var presentation ReviewPRPresentation
	var partial int
	var reviewersJSON, unknownsJSON string
	err := scanner.Scan(
		&presentation.PresentationKey, &presentation.InstallationID, &presentation.Repository,
		&presentation.PRNumber, &presentation.SchemaVersion, &presentation.Title,
		&presentation.Author, &presentation.BaseBranch, &presentation.HeadBranch,
		&presentation.HeadSHA, &presentation.PatchsetID, &presentation.GitLinkState,
		&presentation.FilesCount, &presentation.CommitsCount, &presentation.Additions,
		&presentation.Deletions, &presentation.ReviewStage, &presentation.Decision,
		&presentation.ReviewCount, &presentation.ThreadCount, &presentation.OpenThreadCount,
		&presentation.RiskLevel, &presentation.CollectionStatus, &partial, &reviewersJSON,
		&unknownsJSON, &presentation.RecommendedNextStep, &presentation.GitLinkURL,
		&presentation.SourceFingerprint, &presentation.ContentFingerprint,
		&presentation.SourceCompletedAt, &presentation.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ReviewPRPresentation{}, nil
	}
	if err != nil {
		return ReviewPRPresentation{}, err
	}
	presentation.Partial = partial != 0
	if err := json.Unmarshal([]byte(reviewersJSON), &presentation.Reviewers); err != nil {
		return ReviewPRPresentation{}, fmt.Errorf("decode Review presentation reviewers: %w", err)
	}
	if err := json.Unmarshal([]byte(unknownsJSON), &presentation.Unknowns); err != nil {
		return ReviewPRPresentation{}, fmt.Errorf("decode Review presentation unknowns: %w", err)
	}
	return presentation, nil
}

func writeReviewPRPresentationTx(ctx context.Context, tx *sql.Tx, presentation ReviewPRPresentation) error {
	reviewersJSON, _ := json.Marshal(presentation.Reviewers)
	unknownsJSON, _ := json.Marshal(presentation.Unknowns)
	_, err := tx.ExecContext(ctx, `INSERT INTO review_pr_presentations (
		presentation_key, installation_id, repository, pr_number, schema_version,
		title, author, base_branch, head_branch, head_sha, patchset_id, gitlink_state,
		files_count, commits_count, additions, deletions, review_stage, decision,
		review_count, thread_count, open_thread_count, risk_level, collection_status,
		partial, reviewers_json, unknowns_json, recommended_next_step, gitlink_url,
		source_fingerprint, content_fingerprint, source_completed_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(presentation_key) DO UPDATE SET
		schema_version=excluded.schema_version, title=excluded.title, author=excluded.author,
		base_branch=excluded.base_branch, head_branch=excluded.head_branch,
		head_sha=excluded.head_sha, patchset_id=excluded.patchset_id,
		gitlink_state=excluded.gitlink_state, files_count=excluded.files_count,
		commits_count=excluded.commits_count, additions=excluded.additions,
		deletions=excluded.deletions, review_stage=excluded.review_stage,
		decision=excluded.decision, review_count=excluded.review_count,
		thread_count=excluded.thread_count, open_thread_count=excluded.open_thread_count,
		risk_level=excluded.risk_level, collection_status=excluded.collection_status,
		partial=excluded.partial, reviewers_json=excluded.reviewers_json,
		unknowns_json=excluded.unknowns_json,
		recommended_next_step=excluded.recommended_next_step,
		gitlink_url=excluded.gitlink_url, source_fingerprint=excluded.source_fingerprint,
		content_fingerprint=excluded.content_fingerprint,
		source_completed_at=excluded.source_completed_at, updated_at=excluded.updated_at`,
		presentation.PresentationKey, presentation.InstallationID, presentation.Repository,
		presentation.PRNumber, presentation.SchemaVersion, presentation.Title,
		presentation.Author, presentation.BaseBranch, presentation.HeadBranch,
		presentation.HeadSHA, presentation.PatchsetID, presentation.GitLinkState,
		presentation.FilesCount, presentation.CommitsCount, presentation.Additions,
		presentation.Deletions, presentation.ReviewStage, presentation.Decision,
		presentation.ReviewCount, presentation.ThreadCount, presentation.OpenThreadCount,
		presentation.RiskLevel, presentation.CollectionStatus,
		boolToReviewCollaborationInt(presentation.Partial), string(reviewersJSON),
		string(unknownsJSON), presentation.RecommendedNextStep, presentation.GitLinkURL,
		presentation.SourceFingerprint, presentation.ContentFingerprint,
		presentation.SourceCompletedAt, presentation.UpdatedAt,
	)
	return err
}
