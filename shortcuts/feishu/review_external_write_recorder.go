package feishu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ReviewExternalWriteRecord struct {
	ResourceType  string `json:"resource_type"`
	OperationKind string `json:"operation_kind"`
	RemoteIDHash  string `json:"remote_id_hash,omitempty"`
	ChatIDHash    string `json:"chat_id_hash,omitempty"`
	Timestamp     string `json:"timestamp"`
	Result        string `json:"result"`
}

type ReviewExternalWriteSnapshot struct {
	SchemaVersion string                      `json:"schema_version"`
	Counts        map[string]int              `json:"counts"`
	Records       []ReviewExternalWriteRecord `json:"records"`
}

type ReviewExternalWriteRecorder struct {
	mu      sync.Mutex
	now     func() time.Time
	counts  map[string]int
	records []ReviewExternalWriteRecord
}

func NewReviewExternalWriteRecorder(now func() time.Time) *ReviewExternalWriteRecorder {
	if now == nil {
		now = time.Now
	}
	return &ReviewExternalWriteRecorder{now: now, counts: map[string]int{}}
}

func (r *ReviewExternalWriteRecorder) Record(operation, remoteID, chatID, result string) {
	if r == nil {
		return
	}
	operation = strings.TrimSpace(operation)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[operation]++
	r.records = append(r.records, ReviewExternalWriteRecord{
		ResourceType:  reviewExternalResource(operation),
		OperationKind: operation,
		RemoteIDHash:  hashOptionalReviewIdentifier(remoteID),
		ChatIDHash:    hashOptionalReviewIdentifier(chatID),
		Timestamp:     r.now().UTC().Format(time.RFC3339Nano),
		Result:        strings.TrimSpace(result),
	})
}

func (r *ReviewExternalWriteRecorder) Snapshot() ReviewExternalWriteSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := map[string]int{}
	for _, operation := range []string{
		"gitlink_get", "gitlink_head", "gitlink_post", "gitlink_put", "gitlink_patch", "gitlink_delete",
		"message_create", "message_patch", "reply_send", "base_search", "base_create", "base_update",
		"doc_create", "doc_append", "task_create", "task_patch", "tenant_token",
	} {
		counts[operation] = r.counts[operation]
	}
	return ReviewExternalWriteSnapshot{
		SchemaVersion: "feishu.review-external-write-counts/v1",
		Counts:        counts,
		Records:       append([]ReviewExternalWriteRecord(nil), r.records...),
	}
}

type ReviewValidationRoundTripper struct {
	Base                http.RoundTripper
	Target              string
	Recorder            *ReviewExternalWriteRecorder
	AllowExternalWrites bool
}

func (t *ReviewValidationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, fmt.Errorf("validation request is required")
	}
	operation := classifyReviewValidationRequest(t.Target, request.Method, request.URL.Path)
	if t.Recorder != nil {
		t.Recorder.Record(operation, reviewValidationRemotePathID(request.URL.Path), request.URL.Query().Get("receive_id"), "attempted")
	}
	if strings.EqualFold(t.Target, "gitlink") && request.Method != http.MethodGet && request.Method != http.MethodHead {
		return nil, fmt.Errorf("GitLink validation blocked %s before network transmission", request.Method)
	}
	if strings.EqualFold(t.Target, "feishu") && request.Method != http.MethodGet && request.Method != http.MethodHead && !t.AllowExternalWrites {
		return nil, ErrReviewExternalWritesDenied
	}
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

func classifyReviewValidationRequest(target, method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.ToLower(path)
	if strings.EqualFold(target, "gitlink") {
		return "gitlink_" + strings.ToLower(method)
	}
	switch {
	case strings.Contains(path, "/tenant_access_token"):
		return "tenant_token"
	case strings.HasSuffix(path, "/reply"):
		return "reply_send"
	case strings.Contains(path, "/im/v1/messages") && method == http.MethodPatch:
		return "message_patch"
	case strings.Contains(path, "/im/v1/messages"):
		return "message_create"
	case strings.Contains(path, "/records/search"):
		return "base_search"
	case strings.Contains(path, "/records/") && (method == http.MethodPut || method == http.MethodPatch):
		return "base_update"
	case strings.Contains(path, "/records"):
		return "base_create"
	case strings.Contains(path, "/blocks"):
		return "doc_append"
	case strings.Contains(path, "/documents"):
		return "doc_create"
	case strings.Contains(path, "/tasks/") && method == http.MethodPatch:
		return "task_patch"
	case strings.Contains(path, "/tasks"):
		return "task_create"
	default:
		return "feishu_" + strings.ToLower(method)
	}
}

func reviewExternalResource(operation string) string {
	if index := strings.Index(operation, "_"); index > 0 {
		return operation[:index]
	}
	return operation
}

func reviewValidationRemotePathID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func hashOptionalReviewIdentifier(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return HashReviewValidationIdentifier(value)
}

func MarshalReviewExternalWriteSnapshot(recorder *ReviewExternalWriteRecorder) ([]byte, error) {
	if recorder == nil {
		return nil, fmt.Errorf("external write recorder is required")
	}
	return json.MarshalIndent(recorder.Snapshot(), "", "  ")
}
