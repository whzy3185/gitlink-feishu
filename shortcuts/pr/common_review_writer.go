package pr

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

var commonReviewRefLine = regexp.MustCompile(`(?i)\bRef:\s*RW-[0-9A-F]{6}\b`)
var commonReviewRequestID = regexp.MustCompile(`^RW-[0-9A-F]{6}$`)

type CommonReviewOptions struct {
	Owner, Repository string
	PRNumber          int
	Content           string
	ReviewStatus      string
	ExpectedHead      string
	RequestID         string
	ExpectedActor     string
	DryRun            bool
	BeforePOST        func() error
}

type CommonReviewResult struct {
	Status         string   `json:"status"`
	Repository     string   `json:"repository"`
	PRNumber       int      `json:"pr_number"`
	CurrentHead    string   `json:"current_head"`
	ExpectedHead   string   `json:"expected_head"`
	RequestID      string   `json:"request_id,omitempty"`
	Content        string   `json:"content"`
	ReviewStatus   string   `json:"review_status"`
	ReviewID       string   `json:"review_id,omitempty"`
	Actor          string   `json:"actor,omitempty"`
	Error          string   `json:"error,omitempty"`
	MutationStatus string   `json:"mutation_status"`
	RemoteCommitID *string  `json:"remote_commit_id"`
	Warnings       []string `json:"warnings,omitempty"`
	POSTCount      int      `json:"post_count"`
	WouldWrite     bool     `json:"would_write"`
	Mutated        bool     `json:"mutated"`
}

func BuildCommonReviewContent(content, requestID string) (string, error) {
	content = strings.TrimSpace(commonReviewRefLine.ReplaceAllString(content, ""))
	if content == "" {
		return "", fmt.Errorf("common Review content is required")
	}
	requestID = strings.ToUpper(strings.TrimSpace(requestID))
	if requestID == "" {
		return content, nil
	}
	if !commonReviewRequestID.MatchString(requestID) {
		return "", fmt.Errorf("request ID must match RW-XXXXXX")
	}
	return content + "\n\nRef: " + requestID, nil
}

func ExecuteCommonReview(runtime *common.RuntimeContext, opts CommonReviewOptions) CommonReviewResult {
	opts.ReviewStatus = "common"
	return ExecuteControlledReview(runtime, opts)
}

// ExecuteControlledReview applies the same guarded, single-POST contract to
// GitLink common, approved, and rejected Review decisions.
func ExecuteControlledReview(runtime *common.RuntimeContext, opts CommonReviewOptions) CommonReviewResult {
	status := strings.ToLower(strings.TrimSpace(opts.ReviewStatus))
	if status == "" {
		status = "common"
	}
	result := CommonReviewResult{Repository: opts.Owner + "/" + opts.Repository, PRNumber: opts.PRNumber,
		ExpectedHead: strings.TrimSpace(opts.ExpectedHead), RequestID: strings.ToUpper(strings.TrimSpace(opts.RequestID)),
		MutationStatus: "none", ReviewStatus: status}
	if runtime == nil || runtime.Client == nil || opts.Owner == "" || opts.Repository == "" || opts.PRNumber <= 0 {
		return commonReviewFailure(result, "controlled Review runtime and target are required")
	}
	if status != "common" && status != "approved" && status != "rejected" {
		return commonReviewFailure(result, "controlled Review status must be common, approved, or rejected")
	}
	content, err := BuildCommonReviewContent(opts.Content, result.RequestID)
	if err != nil {
		return commonReviewFailure(result, err.Error())
	}
	result.Content = content
	runtimeCopy := *runtime
	clientCopy := *runtime.Client
	if clientCopy.HTTP == nil {
		clientCopy.HTTP = &http.Client{Timeout: 3 * time.Second}
	} else if clientCopy.HTTP.Timeout == 0 || clientCopy.HTTP.Timeout > 3*time.Second {
		httpCopy := *clientCopy.HTTP
		httpCopy.Timeout, clientCopy.HTTP = 3*time.Second, &httpCopy
	}
	runtimeCopy.Client = &clientCopy
	runtimeCopy.Owner, runtimeCopy.Repo = opts.Owner, opts.Repository
	current, err := workflow.FetchReviewContext(&runtimeCopy, workflow.ReviewContextOptions{
		Owner: opts.Owner, Repo: opts.Repository, Number: opts.PRNumber, IncludePR: true, IncludeVersions: true,
	})
	if err != nil {
		return commonReviewFailure(result, "current pull request is required")
	}
	result.CurrentHead = strings.TrimSpace(current.CurrentHeadSHA)
	if state := strings.ToLower(strings.TrimSpace(current.WorkItem.GitLinkState)); state == "closed" || state == "merged" || result.CurrentHead == "" {
		return commonReviewFailure(result, "pull request must be open with a current head")
	}
	if result.ExpectedHead == "" {
		result.ExpectedHead = result.CurrentHead
	}
	if result.ExpectedHead != result.CurrentHead {
		result.Status, result.Error = "stale", "expected head does not match current head"
		return result
	}
	if opts.DryRun {
		result.Status, result.WouldWrite = "dry_run", true
		return result
	}
	user, err := runtimeCopy.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return commonReviewFailure(result, "resolve active GitLink identity: "+err.Error())
	}
	userMap, _ := user.Data.(map[string]interface{})
	result.Actor = commonReviewString(userMap, "login", "username")
	if result.Actor == "" {
		return commonReviewFailure(result, "active GitLink identity is missing login")
	}
	if opts.ExpectedActor != "" && result.Actor != strings.TrimSpace(opts.ExpectedActor) {
		return commonReviewFailure(result, "active GitLink identity does not match the expected actor")
	}
	if result.RequestID != "" {
		if record := findCommonReview(&runtimeCopy, opts.PRNumber, "", status, content, result.Actor, result.ExpectedHead); record != nil {
			result.Status, result.ReviewID = "duplicate", commonReviewString(record, "id", "review_id")
			result.RemoteCommitID, _ = commonReviewCommit(record)
			return result
		}
	}
	if opts.BeforePOST != nil {
		if err := opts.BeforePOST(); err != nil {
			return commonReviewFailure(result, "persist remote write boundary: "+err.Error())
		}
	}
	result.MutationStatus = "possible"
	envelope, err := runtimeCopy.CallAPI("POST", fmt.Sprintf("/v1/%s/%s/pulls/%d/reviews", opts.Owner, opts.Repository, opts.PRNumber), map[string]interface{}{
		"content": content, "status": status, "commit_id": result.ExpectedHead,
	})
	result.POSTCount = 1
	if err != nil {
		result.Status, result.Error = "unknown", err.Error()
		return result
	}
	result.Mutated, result.MutationStatus = true, "confirmed"
	result.ReviewID = commonReviewID(envelope.Data)
	if result.ReviewID == "" {
		result.Status, result.Error = "unknown", "GitLink POST succeeded without a Review ID"
		return result
	}
	for attempt := 0; attempt < 3; attempt++ {
		record := findCommonReview(&runtimeCopy, opts.PRNumber, result.ReviewID, status, "", "", "")
		if record != nil && strings.EqualFold(commonReviewString(record, "status", "state", "review_status"), status) &&
			strings.TrimSpace(commonReviewString(record, "content", "body", "note", "notes")) == content &&
			(commonReviewActor(record) == "" || commonReviewActor(record) == result.Actor) {
			commit, observed := commonReviewCommit(record)
			result.RemoteCommitID = commit
			if observed && commit == nil {
				result.Warnings = append(result.Warnings, "GitLink returned commit_id=null; preserved without substituting Expected Head")
				result.Status = "verified"
				return result
			}
			if observed && commit != nil && *commit == result.ExpectedHead {
				result.Status = "verified"
				return result
			}
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
		}
	}
	result.Status, result.Error = "unknown", "Review POST succeeded but bounded GET read-back did not verify it"
	return result
}

func commonReviewFailure(result CommonReviewResult, message string) CommonReviewResult {
	result.Status, result.Error, result.MutationStatus = "failed", message, "none"
	return result
}

func findCommonReview(runtime *common.RuntimeContext, number int, id, status, content, actor, head string) map[string]interface{} {
	for _, record := range commonReviewRecords(runtime, number) {
		if id != "" && commonReviewString(record, "id", "review_id") == id {
			return record
		}
		commit, observed := commonReviewCommit(record)
		if id == "" && strings.EqualFold(commonReviewString(record, "status", "state", "review_status"), status) &&
			strings.TrimSpace(commonReviewString(record, "content", "body", "note", "notes")) == content &&
			(commonReviewActor(record) == "" || commonReviewActor(record) == actor) && observed &&
			(commit == nil || *commit == head) {
			return record
		}
	}
	return nil
}
func commonReviewRecords(runtime *common.RuntimeContext, number int) []map[string]interface{} {
	current, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner: runtime.Owner, Repo: runtime.Repo, Number: number, IncludeReviews: true,
	})
	if err != nil {
		return nil
	}
	return current.Reviews
}

func commonReviewString(item map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, exists := item[key]; exists && value != nil {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func commonReviewID(data interface{}) string {
	item, _ := data.(map[string]interface{})
	if item == nil {
		return ""
	}
	if id := commonReviewString(item, "id", "review_id"); id != "" {
		return id
	}
	for _, key := range []string{"review", "data"} {
		if id := commonReviewID(item[key]); id != "" {
			return id
		}
	}
	return ""
}

func commonReviewActor(item map[string]interface{}) string {
	for _, key := range []string{"user", "actor", "reviewer", "author"} {
		nested, _ := item[key].(map[string]interface{})
		if value := commonReviewString(nested, "login", "username"); value != "" {
			return value
		}
	}
	return commonReviewString(item, "login", "username")
}

func commonReviewCommit(item map[string]interface{}) (*string, bool) {
	for _, key := range []string{"commit_id", "commit_sha", "sha"} {
		value, exists := item[key]
		if !exists {
			continue
		}
		if value == nil {
			return nil, true
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		return &text, true
	}
	return nil, false
}
