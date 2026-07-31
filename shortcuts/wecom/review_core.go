package wecom

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/collab"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const reviewCoreResultSchema = "gitlink.review-core-result/v1"

var (
	reviewCoreContextPattern = regexp.MustCompile(`(?i)^(?:查看|刷新)\s*PR\s*#?(\d+)$`)
	reviewCoreDraftPattern   = regexp.MustCompile(`(?i)^生成\s*PR\s*#?(\d+)\s*Review\s*草稿$`)
)

type ReviewCoreResult struct {
	SchemaVersion string      `json:"schema_version"`
	Markdown      string      `json:"markdown"`
	Data          interface{} `json:"data,omitempty"`
	GitLinkWrites int         `json:"gitlink_writes"`
}

type ReviewCoreHandler struct {
	Runtime    *common.RuntimeContext
	Owner      string
	Repo       string
	CoreToken  string
	MaxBody    int64
	JobTimeout time.Duration
}

func newReviewCoreShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-core",
		Description: "Serve the loopback-only GET-only Review Core used by the WeCom bridge",
		Flags: []common.Flag{
			{Name: "listen", Usage: "Loopback listen address", Default: "127.0.0.1:8765"},
			{Name: "repository", Usage: "Bound GitLink repository in owner/repo form", Required: true},
			{Name: "core-token", Usage: "Bridge-to-core bearer token. Defaults to GITLINK_REVIEW_CORE_TOKEN"},
			{Name: "job-timeout-seconds", Usage: "Maximum duration of one GitLink GET-only request", Default: "60"},
		},
		Run: runReviewCore,
	}
}

func runReviewCore(runtime *common.RuntimeContext) error {
	listenAddress := first(runtime.Arg("listen"), "127.0.0.1:8765")
	if err := validateLoopbackListenAddress(listenAddress); err != nil {
		return err
	}
	owner, repo, err := splitRepository(runtime.Arg("repository"))
	if err != nil {
		return err
	}
	coreToken := first(runtime.Arg("core-token"), os.Getenv("GITLINK_REVIEW_CORE_TOKEN"))
	if coreToken == "" {
		return fmt.Errorf("Review Core token is required through --core-token or GITLINK_REVIEW_CORE_TOKEN")
	}
	timeoutSeconds, err := strconv.Atoi(first(runtime.Arg("job-timeout-seconds"), "60"))
	if err != nil || timeoutSeconds < 1 || timeoutSeconds > 300 {
		return fmt.Errorf("--job-timeout-seconds must be between 1 and 300")
	}
	handler := &ReviewCoreHandler{
		Runtime:    runtime,
		Owner:      owner,
		Repo:       repo,
		CoreToken:  coreToken,
		MaxBody:    1 << 20,
		JobTimeout: time.Duration(timeoutSeconds) * time.Second,
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/review/inbound", handler)
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      handler.JobTimeout + 5*time.Second,
		IdleTimeout:       30 * time.Second,
	}
	fmt.Fprintf(os.Stderr, "GitLink Review Core listening on %s (GET-only; GitLink writes: 0)\n", listenAddress)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (h *ReviewCoreHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeReviewCoreError(writer, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if !requestFromLoopback(request.RemoteAddr) {
		writeReviewCoreError(writer, http.StatusForbidden, "loopback_required")
		return
	}
	providedToken := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
	if h.CoreToken == "" || subtle.ConstantTimeCompare([]byte(providedToken), []byte(h.CoreToken)) != 1 {
		writeReviewCoreError(writer, http.StatusUnauthorized, "unauthorized")
		return
	}
	maxBody := h.MaxBody
	if maxBody <= 0 {
		maxBody = 1 << 20
	}
	payload, err := io.ReadAll(io.LimitReader(request.Body, maxBody+1))
	if err != nil || int64(len(payload)) > maxBody {
		writeReviewCoreError(writer, http.StatusBadRequest, "invalid_event")
		return
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var event collab.InboundEvent
	if err := decoder.Decode(&event); err != nil {
		writeReviewCoreError(writer, http.StatusBadRequest, "invalid_event")
		return
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeReviewCoreError(writer, http.StatusBadRequest, "invalid_event")
		return
	}
	if event.SchemaVersion != "gitlink.collab-inbound/v1" ||
		event.Platform != "wecom" ||
		event.EventID == "" ||
		event.ChatID == "" ||
		event.UserID == "" ||
		event.Kind != "message" {
		writeReviewCoreError(writer, http.StatusBadRequest, "invalid_event")
		return
	}
	timeout := h.JobTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(request.Context(), timeout)
	defer cancel()
	result, err := h.execute(ctx, strings.TrimSpace(event.Text))
	if err != nil {
		writeReviewCoreError(writer, http.StatusBadGateway, "read_only_query_failed")
		return
	}
	_ = json.NewEncoder(writer).Encode(result)
}

func (h *ReviewCoreHandler) execute(ctx context.Context, text string) (ReviewCoreResult, error) {
	base := ReviewCoreResult{SchemaVersion: reviewCoreResultSchema, GitLinkWrites: 0}
	switch {
	case strings.EqualFold(text, "help") || text == "帮助":
		base.Markdown = "支持：帮助、查看待审查、查看 PR #编号、刷新 PR #编号、生成 PR #编号 Review 草稿。\nGitLink 写入：0。"
		return base, nil
	case strings.EqualFold(text, "review queue") || text == "查看待审查":
		if h.Runtime == nil {
			return base, fmt.Errorf("Review Core runtime is unavailable")
		}
		result, err := workflow.FetchReviewQueue(h.runtimeForRequest(ctx), workflow.ReviewQueueFetchOptions{
			Owner: h.Owner, Repo: h.Repo, State: "open", StartPage: 1, PageSize: 30, MaxItems: 100,
		})
		if err != nil {
			return base, err
		}
		base.Markdown = formatReviewCoreQueue(result)
		base.Data = result
		return base, nil
	}
	if matches := reviewCoreContextPattern.FindStringSubmatch(text); len(matches) == 2 {
		number, _ := strconv.Atoi(matches[1])
		reviewContext, err := h.fetchContext(ctx, number)
		if err != nil {
			return base, err
		}
		base.Markdown = formatReviewCoreContext(reviewContext, false)
		base.Data = reviewContext
		return base, nil
	}
	if matches := reviewCoreDraftPattern.FindStringSubmatch(text); len(matches) == 2 {
		number, _ := strconv.Atoi(matches[1])
		reviewContext, err := h.fetchContext(ctx, number)
		if err != nil {
			return base, err
		}
		base.Markdown = formatReviewCoreContext(reviewContext, true)
		base.Data = reviewContext
		return base, nil
	}
	return base, fmt.Errorf("unsupported read-only command")
}

func (h *ReviewCoreHandler) fetchContext(ctx context.Context, number int) (workflow.ReviewContext, error) {
	if h.Runtime == nil {
		return workflow.ReviewContext{}, fmt.Errorf("Review Core runtime is unavailable")
	}
	return workflow.FetchReviewContext(h.runtimeForRequest(ctx), workflow.ReviewContextOptions{
		Owner:           h.Owner,
		Repo:            h.Repo,
		Number:          number,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
}

func (h *ReviewCoreHandler) runtimeForRequest(_ context.Context) *common.RuntimeContext {
	if h.Runtime == nil {
		return nil
	}
	runtimeCopy := *h.Runtime
	runtimeCopy.Owner = h.Owner
	runtimeCopy.Repo = h.Repo
	if h.Runtime.Client != nil {
		clientCopy := *h.Runtime.Client
		httpClient := http.DefaultClient
		if clientCopy.HTTP != nil {
			httpClient = clientCopy.HTTP
		}
		httpCopy := *httpClient
		httpCopy.Timeout = h.JobTimeout
		clientCopy.HTTP = &httpCopy
		runtimeCopy.Client = &clientCopy
	}
	return &runtimeCopy
}

func formatReviewCoreQueue(queue workflow.ReviewQueueResult) string {
	lines := []string{
		fmt.Sprintf("GitLink Review Queue：%s", queue.Repository),
		fmt.Sprintf("共 %d 个；高优先级 %d，中优先级 %d。", queue.TotalPRs, queue.HighPriority, queue.MediumPriority),
	}
	for index, item := range queue.Items {
		if index >= 5 {
			break
		}
		lines = append(lines, fmt.Sprintf("- PR #%d %s（%s）", item.Number, item.Title, item.Priority))
	}
	lines = append(lines, "GitLink 写入：0。")
	return strings.Join(lines, "\n")
}

func formatReviewCoreContext(reviewContext workflow.ReviewContext, draft bool) string {
	mode := "上下文"
	if draft {
		mode = "草稿"
	}
	lines := []string{
		fmt.Sprintf("GitLink PR #%d Review %s", reviewContext.PullRequest, mode),
		fmt.Sprintf("仓库：%s", reviewContext.Repository),
		fmt.Sprintf("阶段：%s", reviewContext.WorkItem.ReviewStage),
		fmt.Sprintf("判断：%s", reviewContext.Summary.Decision),
		fmt.Sprintf("完整性：%s", reviewContext.CollectionStatus),
		fmt.Sprintf("当前 head：%s", reviewContext.CurrentHeadSHA),
		fmt.Sprintf("下一步：%s", reviewContext.WorkItem.RecommendedNextStep),
	}
	if draft {
		lines = append(lines, "该内容仅供人工 Review 使用，不会自动提交。")
	}
	lines = append(lines, "GitLink 写入：0。")
	return strings.Join(lines, "\n")
}

func writeReviewCoreError(writer http.ResponseWriter, status int, code string) {
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]interface{}{
		"schema_version": reviewCoreResultSchema,
		"error":          code,
		"gitlink_writes": 0,
	})
}

func validateLoopbackListenAddress(address string) error {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("--listen must be host:port: %w", err)
	}
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("--listen must use a loopback host")
	}
	return nil
}

func requestFromLoopback(remoteAddress string) bool {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func splitRepository(value string) (string, string, error) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(value), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("--repository must use owner/repo")
	}
	return parts[0], parts[1], nil
}
