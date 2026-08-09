package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestGitLinkReviewDataProviderCollectsNormalizedReviewFacts(t *testing.T) {
	var mu sync.Mutex
	methods := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		methods = append(methods, r.Method)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var data interface{}
		path := strings.TrimSuffix(r.URL.Path, ".json")
		switch path {
		case "/v1/test-owner/test-repo":
			data = map[string]interface{}{"is_public": true}
		case "/test-owner/test-repo/pulls/7":
			data = map[string]interface{}{
				"title": "Adapter contract", "state": "open",
				"create_user": map[string]interface{}{"login": "author"},
				"base_branch": "master", "head_branch": "feature/review-data",
			}
		case "/test-owner/test-repo/pulls/7/files":
			data = []interface{}{map[string]interface{}{"filename": "review.go"}}
		case "/v1/test-owner/test-repo/pulls/7/versions":
			data = []interface{}{map[string]interface{}{
				"id": 12, "head_commit_sha": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"base_commit_sha": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"files_count":     1, "commits_count": 2, "add_line_num": 10, "del_line_num": 3,
			}}
		case "/v1/test-owner/test-repo/pulls/7/reviews":
			data = []interface{}{
				map[string]interface{}{
					"id": 20, "status": "rejected", "commit_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"reviewer":   map[string]interface{}{"id": 9, "login": "reviewer"},
					"created_at": "2026-08-08 14:00",
				},
				map[string]interface{}{
					"id": 21, "status": "approved", "commit_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"reviewer":   map[string]interface{}{"id": 9, "login": "reviewer"},
					"created_at": "2026-08-08 15:00",
				},
			}
		case "/v1/test-owner/test-repo/pulls/7/journals":
			if r.URL.Query().Get("is_full") != "true" {
				t.Errorf("is_full = %q, want true", r.URL.Query().Get("is_full"))
			}
			data = []interface{}{map[string]interface{}{
				"id": 31, "review_id": 21, "state": "opened", "note": "Check the edge case.",
				"commit_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"author":    map[string]interface{}{"login": "reviewer"},
			}}
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": data})
	}))
	defer server.Close()

	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	data, err := (GitLinkReviewDataProvider{}).FetchReviewData(context.Background(), runtime, ReviewDataRequest{
		Owner: "test-owner", Repository: "test-repo", PullRequest: 7,
	})
	if err != nil {
		t.Fatalf("FetchReviewData returned error: %v", err)
	}
	if data.CollectionStatus != "complete" || data.Partial {
		t.Fatalf("collection = %s partial=%v", data.CollectionStatus, data.Partial)
	}
	if data.HeadSHA != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || data.Patchset.FilesCount != 1 {
		t.Fatalf("patchset = %+v head=%q", data.Patchset, data.HeadSHA)
	}
	if len(data.ReviewerSummaries) != 1 || data.ReviewerSummaries[0].CurrentDecision != "approved" {
		t.Fatalf("reviewers = %+v", data.ReviewerSummaries)
	}
	if data.Summary.Decision != "approved" || data.Summary.TotalReviews != 2 {
		t.Fatalf("summary = %+v", data.Summary)
	}
	if len(data.Threads) != 1 || data.Threads[0].Content != "Check the edge case." {
		t.Fatalf("threads = %+v", data.Threads)
	}
	if data.SourceFingerprint == "" || !strings.HasPrefix(data.SourceFingerprint, "sha256:") {
		t.Fatalf("source fingerprint = %q", data.SourceFingerprint)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, method := range methods {
		if method != http.MethodGet {
			t.Fatalf("method = %s, want GET-only", method)
		}
	}
}

func TestReviewDataReviewerOrderingIsConservative(t *testing.T) {
	reviews := []ReviewDataReview{
		{ID: "100", Actor: "alice", Status: "rejected", Freshness: "current", CreatedAt: "2026-08-08T07:00:00Z"},
		{ID: "101", Actor: "alice", Status: "approved", Freshness: "current"},
	}
	summaries := summarizeReviewDataReviewers(reviews)
	if len(summaries) != 1 {
		t.Fatalf("summaries = %+v", summaries)
	}
	if summaries[0].CurrentDecision != "unknown" || summaries[0].DecisionOrderKnown {
		t.Fatalf("summary = %+v, want unknown order", summaries[0])
	}
}

func TestReviewDataDecisionDoesNotDependOnOpenThreads(t *testing.T) {
	reviews := []ReviewDataReview{{ID: "1", Actor: "alice", Status: "approved", Freshness: "current"}}
	reviewers := summarizeReviewDataReviewers(reviews)
	needRespond := true
	summary := summarizeReviewData(reviews, reviewers, []ReviewDataThread{{ID: "2", State: "opened", Freshness: "current", NeedRespond: &needRespond, Type: "problem"}})
	if summary.Decision != "approved" {
		t.Fatalf("decision = %q, want approved", summary.Decision)
	}
}

func TestGitLinkReviewDataProviderKeepsStructuredPartialErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, ".json")
		if strings.HasSuffix(path, "/reviews") {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var data interface{} = []interface{}{}
		if strings.HasSuffix(path, "/pulls/3") {
			data = map[string]interface{}{"title": "Partial response", "state": "open", "head_commit_sha": "abcdef0123456789"}
		} else if path == "/v1/test-owner/test-repo" {
			data = map[string]interface{}{"is_public": true}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": data})
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	data, err := (GitLinkReviewDataProvider{}).FetchReviewData(context.Background(), runtime, ReviewDataRequest{Owner: "test-owner", Repository: "test-repo", PullRequest: 3})
	if err != nil {
		t.Fatalf("FetchReviewData returned error: %v", err)
	}
	if !data.Partial || data.CollectionStatus != "partial" || len(data.FetchErrors) != 1 {
		t.Fatalf("data = status=%s partial=%v errors=%+v", data.CollectionStatus, data.Partial, data.FetchErrors)
	}
	if data.FetchErrors[0].Section != "reviews" || data.FetchErrors[0].StatusCode != http.StatusBadGateway || !data.FetchErrors[0].Retryable {
		t.Fatalf("fetch error = %+v", data.FetchErrors[0])
	}
}

func TestGitLinkReviewDataProviderRequiresPullRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, ".json")
		if strings.HasSuffix(path, "/pulls/9") {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"is_public": true}})
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()}}
	data, err := (GitLinkReviewDataProvider{}).FetchReviewData(context.Background(), runtime, ReviewDataRequest{Owner: "test-owner", Repository: "test-repo", PullRequest: 9})
	if err == nil || !strings.Contains(err.Error(), "fetch required pull request") {
		t.Fatalf("error = %v, want required PR error", err)
	}
	if data.CollectionStatus != "failed" || !data.Partial {
		t.Fatalf("data = status=%s partial=%v", data.CollectionStatus, data.Partial)
	}
}

func TestSafeReviewDataErrorRedactsCredentials(t *testing.T) {
	message := safeReviewDataError(assertReviewDataError("request failed token=abcdefghijklmnopqrstuvwxyz"))
	if strings.Contains(message, "abcdefghijklmnopqrstuvwxyz") || !strings.Contains(message, "<redacted>") {
		t.Fatalf("message = %q", message)
	}
}

type assertReviewDataError string

func (e assertReviewDataError) Error() string { return string(e) }
