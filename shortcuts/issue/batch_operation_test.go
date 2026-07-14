package issue

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// ---------------------------------------------------------------------------
// RunBatch 核心行为测试
// ---------------------------------------------------------------------------

// TestRunBatchDryRunDoesNotCallFn 验证 dry-run 模式下 fn 不被调用，
// 且 summary 正确反映所有 issue 为 succeeded/dry_run。
func TestRunBatchDryRunDoesNotCallFn(t *testing.T) {
	t.Setenv("GITLINK_CONFIRM_BATCH", "") // 隔离环境变量

	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// dry-run 不应产生任何 HTTP 请求
		t.Fatalf("unexpected HTTP request in dry-run: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)
	numbers := []string{"1", "2"}

	fn := func(_ *common.RuntimeContext, _ string) error {
		t.Fatal("fn should not be called in dry-run mode")
		return nil
	}

	opts := BatchOptions{DryRun: true, Confirm: false}
	summary, err := RunBatch(ctx, numbers, "close", opts, fn)
	if err != nil {
		t.Fatalf("RunBatch dry-run returned error: %v", err)
	}

	common.AssertEqual(t, summary.Total, 2)
	common.AssertEqual(t, summary.Succeeded, 2)
	common.AssertEqual(t, summary.DryRun, true)
	common.AssertEqual(t, summary.Failed, 0)

	for _, r := range summary.Results {
		common.AssertEqual(t, r.Status, "dry_run")
	}
}

// TestRunBatchRequiresConfirm 验证非 dry-run 且无 confirm 时返回确认错误。
func TestRunBatchRequiresConfirm(t *testing.T) {
	t.Setenv("GITLINK_CONFIRM_BATCH", "")

	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)

	fn := func(_ *common.RuntimeContext, _ string) error {
		t.Fatal("fn should not be called without confirm")
		return nil
	}

	opts := BatchOptions{DryRun: false, Confirm: false}
	_, err := RunBatch(ctx, []string{"1"}, "close", opts, fn)
	if err == nil {
		t.Fatal("expected error when confirm is required but not provided")
	}
	if !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("error should mention 'confirm', got: %v", err)
	}
}

// TestRunBatchWithConfirm 验证 confirm=true 时 fn 被正常调用。
func TestRunBatchWithConfirm(t *testing.T) {
	t.Setenv("GITLINK_CONFIRM_BATCH", "")

	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// fn 不做网络请求，不需要 mock
		t.Fatalf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)
	numbers := []string{"1", "2", "3"}

	var callCount int32
	fn := func(_ *common.RuntimeContext, number string) error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	opts := BatchOptions{DryRun: false, Confirm: true}
	summary, err := RunBatch(ctx, numbers, "close", opts, fn)
	if err != nil {
		t.Fatalf("RunBatch with confirm returned error: %v", err)
	}

	common.AssertEqual(t, int(atomic.LoadInt32(&callCount)), 3)
	common.AssertEqual(t, summary.Total, 3)
	common.AssertEqual(t, summary.Succeeded, 3)
	common.AssertEqual(t, summary.Failed, 0)
}

// TestRunBatchMaxTruncation 验证 --max 截断行为。
func TestRunBatchMaxTruncation(t *testing.T) {
	t.Setenv("GITLINK_CONFIRM_BATCH", "")

	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)
	numbers := []string{"1", "2", "3"}

	fn := func(_ *common.RuntimeContext, _ string) error {
		return nil
	}

	opts := BatchOptions{DryRun: true, MaxItems: 2}
	summary, err := RunBatch(ctx, numbers, "close", opts, fn)
	if err == nil {
		t.Fatal("expected error when results are truncated")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error should mention 'truncated', got: %v", err)
	}

	common.AssertEqual(t, summary.Total, 2)     // 截断后为 2
	common.AssertEqual(t, summary.Truncated, true)
	common.AssertEqual(t, summary.Succeeded, 2) // dry-run 全部 succeeded
}

// TestRunBatchRecordsFailures 验证 fn 返回错误时记录为 failed。
func TestRunBatchRecordsFailures(t *testing.T) {
	t.Setenv("GITLINK_CONFIRM_BATCH", "")

	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)
	numbers := []string{"1", "2"}

	var callCount int32
	fn := func(_ *common.RuntimeContext, number string) error {
		n := atomic.AddInt32(&callCount, 1)
		if n == 2 {
			return fmt.Errorf("simulated failure for issue %s", number)
		}
		return nil
	}

	opts := BatchOptions{DryRun: false, Confirm: true}
	summary, err := RunBatch(ctx, numbers, "close", opts, fn)
	if err == nil {
		t.Fatal("expected error when some issues fail")
	}

	common.AssertEqual(t, summary.Total, 2)
	common.AssertEqual(t, summary.Succeeded, 1)
	common.AssertEqual(t, summary.Failed, 1)
	common.AssertEqual(t, summary.Results[0].Status, "success")
	common.AssertEqual(t, summary.Results[1].Status, "failed")
	if summary.Results[1].Error == "" {
		t.Fatal("failed result should have an error message")
	}
}

// ---------------------------------------------------------------------------
// patchIssue 测试（httptest mock）
// ---------------------------------------------------------------------------

// TestPatchIssueMergesExtraFields 验证 patchIssue 将 extraFields 合并到 PATCH body，
// 同时保留 subject 和 description。
func TestPatchIssueMergesExtraFields(t *testing.T) {
	var patchPayload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			common.WriteJSON(t, w, map[string]interface{}{
				"subject":     "Original Title",
				"description": "Original Desc",
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issues/42.json":
			patchPayload = common.DecodeJSON(t, r)
			common.WriteJSON(t, w, patchPayload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ctx := newBatchTestCtx(t, server, nil)
	err := patchIssue(ctx, "42", map[string]interface{}{"status_id": closeIssueStatusID}, "close")
	if err != nil {
		t.Fatalf("patchIssue failed: %v", err)
	}

	common.AssertEqual(t, patchPayload["subject"], "Original Title")
	common.AssertEqual(t, patchPayload["description"], "Original Desc")
	common.AssertEqual(t, patchPayload["status_id"], float64(5))
}

// ---------------------------------------------------------------------------
// 纯函数单元测试
// ---------------------------------------------------------------------------

// TestMergeLabelIDs 验证 mergeLabelIDs 去重合并逻辑。
func TestMergeLabelIDs(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		want []int
	}{
		{"去重合并", []int{1, 2}, []int{2, 3}, []int{1, 2, 3}},
		{"existing 为 nil", nil, []int{1}, []int{1}},
		{"new 为 nil", []int{1}, nil, []int{1}},
		{"两者都为 nil", nil, nil, nil},
		{"完全重复", []int{1, 2}, []int{1, 2}, []int{1, 2}},
		{"existing 为空", []int{}, []int{1, 2}, []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeLabelIDs(tt.a, tt.b)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("mergeLabelIDs(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestRemoveLabelIDs 验证 removeLabelIDs 移除逻辑。
func TestRemoveLabelIDs(t *testing.T) {
	tests := []struct {
		name     string
		existing []int
		remove   []int
		want     []int
	}{
		{"移除中间元素", []int{1, 2, 3}, []int{2}, []int{1, 3}},
		{"移除不存在的忽略", []int{1, 2}, []int{3}, []int{1, 2}},
		{"全部移除", []int{1, 2}, []int{1, 2}, []int{}},
		{"existing 为空", []int{}, []int{1}, []int{}},
		{"remove 为空", []int{1, 2}, []int{}, []int{1, 2}},
		{"两者都为空", []int{}, []int{}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeLabelIDs(tt.existing, tt.remove)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("removeLabelIDs(%v, %v) = %v, want %v", tt.existing, tt.remove, got, tt.want)
			}
		})
	}
}

// TestNormalizeIssueStatus 验证 normalizeIssueStatus 状态映射。
func TestNormalizeIssueStatus(t *testing.T) {
	tests := []struct {
		input string
		want  interface{}
		err   bool
	}{
		{"open", 1, false},
		{"closed", 5, false},
		{"OPEN", 1, false},
		{"Closed", 5, false},
		{"1", 1, false},          // 数字字符串
		{"5", 5, false},          // 数字字符串
		{"invalid", nil, true},   // 无效输入应返回错误
		{"", nil, true},          // 空字符串应返回错误
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeIssueStatus(tt.input)
			if tt.err {
				if err == nil {
					t.Fatalf("normalizeIssueStatus(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeIssueStatus(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeIssueStatus(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseBatchOptions 测试
// ---------------------------------------------------------------------------

// TestParseBatchOptions 验证 parseBatchOptions 从 ctx.Args 正确解析各选项。
func TestParseBatchOptions(t *testing.T) {
	t.Run("完整参数解析", func(t *testing.T) {
		ctx := &common.RuntimeContext{
			Args: map[string]string{
				"dry-run": "true",
				"confirm": "true",
				"max":     "5",
				"delay":   "100",
			},
		}
		opts := parseBatchOptions(ctx)
		common.AssertEqual(t, opts.DryRun, true)
		common.AssertEqual(t, opts.Confirm, true)
		common.AssertEqual(t, opts.MaxItems, 5)
		common.AssertEqual(t, opts.DelayMs, 100)
	})

	t.Run("空参数使用默认值", func(t *testing.T) {
		ctx := &common.RuntimeContext{
			Args: map[string]string{},
		}
		opts := parseBatchOptions(ctx)
		common.AssertEqual(t, opts.DryRun, false)
		common.AssertEqual(t, opts.Confirm, false)
		common.AssertEqual(t, opts.MaxItems, defaultBatchMaxItems)
		common.AssertEqual(t, opts.DelayMs, defaultBatchDelayMs)
	})

	t.Run("无效 max 值使用默认值", func(t *testing.T) {
		ctx := &common.RuntimeContext{
			Args: map[string]string{
				"max": "not-a-number",
			},
		}
		opts := parseBatchOptions(ctx)
		common.AssertEqual(t, opts.MaxItems, defaultBatchMaxItems)
	})

	t.Run("无效 delay 值使用默认值", func(t *testing.T) {
		ctx := &common.RuntimeContext{
			Args: map[string]string{
				"delay": "abc",
			},
		}
		opts := parseBatchOptions(ctx)
		common.AssertEqual(t, opts.DelayMs, defaultBatchDelayMs)
	})
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// newBatchTestCtx 构造用于 batch 测试的 RuntimeContext。
// Args 如果为 nil，则使用空 map。
func newBatchTestCtx(t *testing.T, server *httptest.Server, args map[string]string) *common.RuntimeContext {
	t.Helper()
	if args == nil {
		args = map[string]string{}
	}
	return &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
}
