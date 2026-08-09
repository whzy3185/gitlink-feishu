package feishu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const reviewDataSchemaVersion = "feishu.review-data/v1"

// ReviewDataProvider is the read-only boundary between the Feishu integration
// and GitLink pull request APIs. Implementations must not perform mutations.
type ReviewDataProvider interface {
	FetchReviewData(context.Context, *common.RuntimeContext, ReviewDataRequest) (ReviewData, error)
}

type ReviewDataRequest struct {
	Owner        string
	Repository   string
	PullRequest  int
	VersionLimit int
	ReviewLimit  int
	ThreadLimit  int
}

// ReviewData contains only the GitLink facts consumed by the review gateway.
// It intentionally does not expose workflow ReviewContext, ReviewQueue, or
// WorkItem product semantics.
type ReviewData struct {
	SchemaVersion     string                      `json:"schema_version"`
	Repository        string                      `json:"repository"`
	PullRequest       int                         `json:"pull_request"`
	Title             string                      `json:"title,omitempty"`
	Author            string                      `json:"author,omitempty"`
	State             string                      `json:"state"`
	BaseBranch        string                      `json:"base_branch,omitempty"`
	HeadBranch        string                      `json:"head_branch,omitempty"`
	HeadSHA           string                      `json:"head_sha,omitempty"`
	VersionID         string                      `json:"version_id,omitempty"`
	GitLinkURL        string                      `json:"gitlink_url,omitempty"`
	RepositoryPublic  *bool                       `json:"repository_public,omitempty"`
	Patchset          ReviewDataPatchset          `json:"patchset"`
	Reviews           []ReviewDataReview          `json:"reviews"`
	ReviewerSummaries []ReviewDataReviewerSummary `json:"reviewer_summaries"`
	Threads           []ReviewDataThread          `json:"threads"`
	Summary           ReviewDataSummary           `json:"summary"`
	CollectionStatus  string                      `json:"collection_status"`
	Partial           bool                        `json:"partial"`
	SectionStatuses   []ReviewDataSectionStatus   `json:"section_statuses"`
	FetchErrors       []ReviewDataFetchError      `json:"fetch_errors"`
	Unknowns          []string                    `json:"unknowns"`
	SourceFingerprint string                      `json:"source_fingerprint"`
}

type ReviewDataPatchset struct {
	ID           string `json:"id,omitempty"`
	HeadSHA      string `json:"head_sha,omitempty"`
	BaseSHA      string `json:"base_sha,omitempty"`
	StartSHA     string `json:"start_sha,omitempty"`
	FilesCount   int    `json:"files_count"`
	CommitsCount int    `json:"commits_count"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type ReviewDataReview struct {
	ID         string `json:"id,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	Actor      string `json:"actor,omitempty"`
	Status     string `json:"status"`
	CommitID   string `json:"commit_id,omitempty"`
	Freshness  string `json:"freshness"`
	Content    string `json:"content,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	SourceType string `json:"source_type"`
}

type ReviewDataReviewerSummary struct {
	ReviewerKey           string            `json:"reviewer_key"`
	ActorID               string            `json:"actor_id,omitempty"`
	Actor                 string            `json:"actor,omitempty"`
	CurrentDecision       string            `json:"current_decision"`
	CurrentCount          int               `json:"current_count"`
	OutdatedCount         int               `json:"outdated_count"`
	UnknownCount          int               `json:"unknown_count"`
	DecisionOrderKnown    bool              `json:"decision_order_known"`
	LatestEffectiveReview *ReviewDataReview `json:"latest_effective_review,omitempty"`
}

type ReviewDataThread struct {
	ID            string `json:"id,omitempty"`
	ReviewID      string `json:"review_id,omitempty"`
	ParentID      string `json:"parent_id,omitempty"`
	AuthorID      string `json:"author_id,omitempty"`
	Author        string `json:"author,omitempty"`
	Content       string `json:"content,omitempty"`
	Type          string `json:"type,omitempty"`
	State         string `json:"state"`
	NeedRespond   *bool  `json:"need_respond,omitempty"`
	Path          string `json:"path,omitempty"`
	LineCode      string `json:"line_code,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`
	Freshness     string `json:"freshness"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	UnknownParent bool   `json:"unknown_parent,omitempty"`
}

type ReviewDataSummary struct {
	ReviewStatus    string `json:"review_status"`
	ReviewFreshness string `json:"review_freshness"`
	Decision        string `json:"decision"`
	TotalReviews    int    `json:"total_reviews"`
	CurrentReviews  int    `json:"current_reviews"`
	OutdatedReviews int    `json:"outdated_reviews"`
	UnknownReviews  int    `json:"unknown_reviews"`
	TotalThreads    int    `json:"total_threads"`
	OpenThreads     int    `json:"open_threads"`
}

type ReviewDataSectionStatus struct {
	Section   string `json:"section"`
	Status    string `json:"status"`
	Required  bool   `json:"required"`
	ItemCount int    `json:"item_count,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type ReviewDataFetchError struct {
	Section    string `json:"section"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Code       string `json:"code"`
	StatusCode int    `json:"status_code,omitempty"`
	Retryable  bool   `json:"retryable"`
	Message    string `json:"message"`
}

// GitLinkReviewDataProvider collects the bounded set of GitLink GET endpoints
// needed by the Feishu review gateway.
type GitLinkReviewDataProvider struct{}

var _ ReviewDataProvider = GitLinkReviewDataProvider{}

func (GitLinkReviewDataProvider) FetchReviewData(
	ctx context.Context,
	runtime *common.RuntimeContext,
	request ReviewDataRequest,
) (ReviewData, error) {
	owner := strings.TrimSpace(request.Owner)
	repository := strings.TrimSpace(request.Repository)
	if owner == "" || repository == "" {
		return ReviewData{}, fmt.Errorf("review data requires owner and repository")
	}
	if request.PullRequest <= 0 {
		return ReviewData{}, fmt.Errorf("review data requires a positive pull request number")
	}
	if runtime == nil || runtime.Client == nil {
		return ReviewData{}, fmt.Errorf("review data requires a GitLink runtime")
	}
	request.VersionLimit = boundedReviewDataLimit(request.VersionLimit)
	request.ReviewLimit = boundedReviewDataLimit(request.ReviewLimit)
	request.ThreadLimit = boundedReviewDataLimit(request.ThreadLimit)

	data := ReviewData{
		SchemaVersion:     reviewDataSchemaVersion,
		Repository:        owner + "/" + repository,
		PullRequest:       request.PullRequest,
		State:             "unknown",
		Reviews:           []ReviewDataReview{},
		ReviewerSummaries: []ReviewDataReviewerSummary{},
		Threads:           []ReviewDataThread{},
		SectionStatuses: []ReviewDataSectionStatus{
			{Section: "repository", Status: "pending"},
			{Section: "pull_request", Status: "pending", Required: true},
			{Section: "files", Status: "pending", Required: true, Limit: 100},
			{Section: "versions", Status: "pending", Required: true, Limit: request.VersionLimit},
			{Section: "reviews", Status: "pending", Required: true, Limit: request.ReviewLimit},
			{Section: "threads", Status: "pending", Required: true, Limit: request.ThreadLimit},
		},
		FetchErrors: []ReviewDataFetchError{},
		Unknowns:    []string{},
	}

	if err := ctx.Err(); err != nil {
		return data, err
	}
	repoPath := fmt.Sprintf("/v1/%s/%s", owner, repository)
	if item, err := fetchReviewDataObject(runtime, repoPath); err != nil {
		recordReviewDataError(&data, "repository", repoPath, err)
	} else {
		if value, ok := reviewDataBool(item, "is_public", "public"); ok {
			data.RepositoryPublic = &value
		}
		markReviewDataSection(&data, "repository", 1, 0)
	}

	prPath := fmt.Sprintf("/%s/%s/pulls/%d", owner, repository, request.PullRequest)
	pr, err := fetchReviewDataObject(runtime, prPath)
	if err != nil {
		recordReviewDataError(&data, "pull_request", prPath, err)
		finalizeReviewData(&data)
		return data, fmt.Errorf("fetch required pull request: %s", safeReviewDataError(err))
	}
	populateReviewDataPR(&data, pr)
	markReviewDataSection(&data, "pull_request", 1, 0)

	filesPath := prPath + "/files"
	files, err := fetchReviewDataList(runtime, filesPath, nil, 100)
	if err != nil {
		recordReviewDataError(&data, "files", filesPath, err)
	} else {
		markReviewDataSection(&data, "files", len(files), 100)
		if data.Patchset.FilesCount == 0 {
			data.Patchset.FilesCount = len(files)
		}
	}

	versionPath := repoPath + fmt.Sprintf("/pulls/%d/versions", request.PullRequest)
	versions, err := fetchReviewDataList(runtime, versionPath, nil, request.VersionLimit)
	if err != nil {
		recordReviewDataError(&data, "versions", versionPath, err)
	} else {
		data.Patchset = normalizeReviewDataPatchset(versions, data.Patchset)
		data.HeadSHA = firstReviewDataValue(data.Patchset.HeadSHA, data.HeadSHA)
		data.VersionID = firstReviewDataValue(data.Patchset.ID, data.VersionID)
		markReviewDataSection(&data, "versions", len(versions), request.VersionLimit)
	}

	reviewPath := repoPath + fmt.Sprintf("/pulls/%d/reviews", request.PullRequest)
	reviews, err := fetchReviewDataList(runtime, reviewPath, nil, request.ReviewLimit)
	if err != nil {
		recordReviewDataError(&data, "reviews", reviewPath, err)
	} else {
		data.Reviews = normalizeReviewDataReviews(reviews, data.HeadSHA)
		markReviewDataSection(&data, "reviews", len(reviews), request.ReviewLimit)
	}

	threadPath := repoPath + fmt.Sprintf("/pulls/%d/journals", request.PullRequest)
	threads, err := fetchReviewDataList(runtime, threadPath, url.Values{"is_full": []string{"true"}}, request.ThreadLimit)
	if err != nil {
		recordReviewDataError(&data, "threads", threadPath, err)
	} else {
		data.Threads = normalizeReviewDataThreads(threads, data.HeadSHA)
		markReviewDataSection(&data, "threads", len(threads), request.ThreadLimit)
	}

	finalizeReviewData(&data)
	return data, nil
}

func boundedReviewDataLimit(value int) int {
	if value <= 0 || value > 100 {
		return 100
	}
	return value
}

func fetchReviewDataObject(runtime *common.RuntimeContext, path string) (map[string]interface{}, error) {
	envelope, err := runtime.CallAPI("GET", path, nil)
	if err != nil {
		return nil, err
	}
	if item := reviewDataObject(envelope.Data); item != nil {
		return item, nil
	}
	return nil, fmt.Errorf("GitLink response did not contain an object")
}

func fetchReviewDataList(runtime *common.RuntimeContext, path string, query url.Values, limit int) ([]map[string]interface{}, error) {
	values := url.Values{}
	for key, entries := range query {
		values[key] = append([]string(nil), entries...)
	}
	values.Set("page", "1")
	values.Set("limit", strconv.Itoa(limit))
	envelope, err := runtime.CallAPIWithQuery("GET", path, values)
	if err != nil {
		return nil, err
	}
	items := reviewDataList(envelope.Data)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func recordReviewDataError(data *ReviewData, section, path string, err error) {
	entry := ReviewDataFetchError{
		Section: section, Method: "GET", Path: path, Code: "request_failed",
		Retryable: true, Message: safeReviewDataError(err),
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		entry.StatusCode = apiErr.StatusCode
		entry.Code = fmt.Sprint(apiErr.Code)
		entry.Retryable = apiErr.StatusCode == 408 || apiErr.StatusCode == 429 || apiErr.StatusCode >= 500
	}
	data.FetchErrors = append(data.FetchErrors, entry)
	for index := range data.SectionStatuses {
		if data.SectionStatuses[index].Section == section {
			data.SectionStatuses[index].Status = "failed"
			return
		}
	}
}

func markReviewDataSection(data *ReviewData, section string, count, limit int) {
	status := "loaded"
	if count == 0 {
		status = "empty"
	} else if limit > 0 && count >= limit {
		status = "sampled"
	}
	for index := range data.SectionStatuses {
		if data.SectionStatuses[index].Section == section {
			data.SectionStatuses[index].Status = status
			data.SectionStatuses[index].ItemCount = count
			return
		}
	}
}

func finalizeReviewData(data *ReviewData) {
	data.ReviewerSummaries = summarizeReviewDataReviewers(data.Reviews)
	data.Summary = summarizeReviewData(data.Reviews, data.ReviewerSummaries, data.Threads)
	for _, review := range data.Reviews {
		if review.Freshness == "unknown" {
			data.Unknowns = append(data.Unknowns, "review_commit_binding_unknown")
			break
		}
	}
	for _, thread := range data.Threads {
		if thread.Freshness == "unknown" {
			data.Unknowns = append(data.Unknowns, "review_thread_commit_binding_unknown")
		}
		if thread.UnknownParent {
			data.Unknowns = append(data.Unknowns, "orphan_review_thread_reply")
		}
	}
	if data.HeadSHA == "" {
		data.Unknowns = append(data.Unknowns, "current_head_sha_not_returned")
	}
	requiredComplete := true
	for _, section := range data.SectionStatuses {
		if section.Required && section.Status != "loaded" && section.Status != "empty" {
			requiredComplete = false
		}
	}
	data.Unknowns = sortedReviewDataStrings(data.Unknowns)
	switch {
	case len(data.FetchErrors) > 0 && !reviewDataSectionSucceeded(data, "pull_request"):
		data.CollectionStatus = "failed"
	case len(data.FetchErrors) > 0 || !requiredComplete || len(data.Unknowns) > 0:
		data.CollectionStatus = "partial"
	default:
		data.CollectionStatus = "complete"
	}
	data.Partial = data.CollectionStatus != "complete"
	sort.Slice(data.SectionStatuses, func(i, j int) bool { return data.SectionStatuses[i].Section < data.SectionStatuses[j].Section })
	sort.Slice(data.FetchErrors, func(i, j int) bool { return data.FetchErrors[i].Section < data.FetchErrors[j].Section })
	data.SourceFingerprint = reviewDataFingerprint(*data)
}

func reviewDataSectionSucceeded(data *ReviewData, section string) bool {
	for _, status := range data.SectionStatuses {
		if status.Section == section {
			return status.Status == "loaded" || status.Status == "empty"
		}
	}
	return false
}

func reviewDataFingerprint(data ReviewData) string {
	type fetchErrorFingerprint struct {
		Section string `json:"section"`
		Code    string `json:"code"`
		Status  int    `json:"status"`
	}
	errorsForHash := make([]fetchErrorFingerprint, 0, len(data.FetchErrors))
	for _, item := range data.FetchErrors {
		errorsForHash = append(errorsForHash, fetchErrorFingerprint{
			Section: item.Section,
			Code:    item.Code,
			Status:  item.StatusCode,
		})
	}
	payload := struct {
		Schema     string                      `json:"schema"`
		Repository string                      `json:"repository"`
		Number     int                         `json:"number"`
		Head       string                      `json:"head"`
		Patchset   ReviewDataPatchset          `json:"patchset"`
		Reviews    []ReviewDataReview          `json:"reviews"`
		Reviewers  []ReviewDataReviewerSummary `json:"reviewers"`
		Threads    []ReviewDataThread          `json:"threads"`
		Status     string                      `json:"status"`
		Errors     []fetchErrorFingerprint     `json:"errors"`
	}{
		Schema:     reviewDataSchemaVersion,
		Repository: data.Repository,
		Number:     data.PullRequest,
		Head:       data.HeadSHA,
		Patchset:   data.Patchset,
		Reviews:    data.Reviews,
		Reviewers:  data.ReviewerSummaries,
		Threads:    data.Threads,
		Status:     data.CollectionStatus,
		Errors:     errorsForHash,
	}
	encoded, _ := json.Marshal(payload)
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}

var reviewDataSensitivePattern = regexp.MustCompile(`(?i)(authorization|cookie|token|secret|password)=?[^\s&]+`)

func safeReviewDataError(err error) string {
	if err == nil {
		return ""
	}
	message := reviewDataSensitivePattern.ReplaceAllString(err.Error(), "$1=<redacted>")
	if len(message) > 300 {
		message = message[:300]
	}
	return message
}
