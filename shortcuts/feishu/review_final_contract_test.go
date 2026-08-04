package feishu

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type reviewValidationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f reviewValidationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRealValidationRequiresBothExplicitSwitches(t *testing.T) {
	config := validReviewRealValidationConfig(t)
	config.RealValidation = false
	if err := config.Validate(false); !errors.Is(err, ErrReviewRealValidationDisabled) {
		t.Fatalf("disabled error = %v", err)
	}
	config.RealValidation = true
	config.ExternalWrites = false
	if err := config.Validate(true); !errors.Is(err, ErrReviewExternalWritesDenied) {
		t.Fatalf("write confirmation error = %v", err)
	}
	config.ExternalWrites = true
	if err := config.Validate(true); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}

func TestGitLinkValidationBlocksWritesBeforeNetwork(t *testing.T) {
	called := 0
	recorder := NewReviewExternalWriteRecorder(func() time.Time { return time.Unix(0, 0) })
	transport := &ReviewValidationRoundTripper{
		Target:   "gitlink",
		Recorder: recorder,
		Base: reviewValidationRoundTripFunc(func(*http.Request) (*http.Response, error) {
			called++
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}),
	}
	request, _ := http.NewRequest(http.MethodPost, "https://gitlink.example/api/owner/repo/pulls/1/reviews", strings.NewReader(`{}`))
	if _, err := transport.RoundTrip(request); err == nil || !strings.Contains(err.Error(), "before network") {
		t.Fatalf("write guard error = %v", err)
	}
	if called != 0 || recorder.Snapshot().Counts["gitlink_post"] != 1 {
		t.Fatalf("called=%d counts=%#v", called, recorder.Snapshot().Counts)
	}
	request, _ = http.NewRequest(http.MethodGet, "https://gitlink.example/api/owner/repo/pulls/1", nil)
	if _, err := transport.RoundTrip(request); err != nil || called != 1 {
		t.Fatalf("read err=%v called=%d", err, called)
	}
}

func TestExternalWriteRecorderUsesHashesAndExactCounters(t *testing.T) {
	recorder := NewReviewExternalWriteRecorder(func() time.Time { return time.Date(2026, 8, 4, 1, 2, 3, 0, time.UTC) })
	recorder.Record("message_create", "om-sensitive", "oc-sensitive", "success")
	recorder.Record("base_search", "record-sensitive", "", "success")
	snapshot := recorder.Snapshot()
	if snapshot.Counts["message_create"] != 1 || snapshot.Counts["base_search"] != 1 || snapshot.Counts["message_patch"] != 0 {
		t.Fatalf("counts = %#v", snapshot.Counts)
	}
	encoded, err := MarshalReviewExternalWriteSnapshot(recorder)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, sensitive := range []string{"om-sensitive", "oc-sensitive", "record-sensitive"} {
		if strings.Contains(text, sensitive) {
			t.Fatalf("snapshot leaked %q: %s", sensitive, text)
		}
	}
}

func TestNewReviewValidationRunID(t *testing.T) {
	runID, err := NewReviewValidationRunID(time.Date(2026, 8, 4, 5, 6, 7, 0, time.FixedZone("test", 8*60*60)), "8cd9dba")
	if err != nil || runID != "review-final-20260803T210607Z-8cd9dba" {
		t.Fatalf("runID=%q err=%v", runID, err)
	}
}

func validReviewRealValidationConfig(t *testing.T) ReviewRealValidationConfig {
	t.Helper()
	t.Setenv("TEST_REVIEW_GITLINK_TOKEN", "secret-not-serialized")
	return ReviewRealValidationConfig{
		RunID:             "review-final-20260804T010203Z-8cd9dba",
		GitLinkBaseURL:    "https://www.gitlink.org.cn/api",
		GitLinkCredential: "env:TEST_REVIEW_GITLINK_TOKEN",
		Repository:        "Gitlink/gitlink-cli",
		PRNumber:          431,
		AppID:             "cli_test",
		AppSecret:         "secret",
		ChatA:             "oc_test_a",
		ChatB:             "oc_test_b",
		WebhookSecret:     "secret",
		WebhookURL:        "https://review-test.example/webhook",
		BaseAppToken:      "app_test",
		TableID:           "tbl_test",
		DocumentFolder:    "fld_test",
		StateDatabase:     "review-final-test.db",
		BackupDirectory:   "review-final-test-backups",
		ResourcePrefix:    "review-final-stage6",
		AdminAddress:      "127.0.0.1:18081",
		WebhookAddress:    "127.0.0.1:18082",
		RealValidation:    true,
		ExternalWrites:    true,
	}
}
