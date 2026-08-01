package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const reviewAgentRunSchema = "review.agent-run/v1"

type ReviewAgentInvocation struct {
	SchemaVersion     string          `json:"schema_version"`
	RunID             string          `json:"run_id"`
	Repository        string          `json:"repository"`
	PRNumber          int             `json:"pr_number"`
	HeadSHA           string          `json:"head_sha"`
	SourceFingerprint string          `json:"source_fingerprint"`
	Mode              string          `json:"mode"`
	Task              ReviewAgentTask `json:"task"`
	GitLinkWrites     int             `json:"gitlink_writes"`
}

type ReviewAgentRun struct {
	SchemaVersion string                  `json:"schema_version"`
	Status        string                  `json:"status"`
	RunID         string                  `json:"run_id"`
	Plan          ReviewAgentPlan         `json:"plan"`
	Assessments   []ReviewAgentAssessment `json:"assessments"`
	Errors        []string                `json:"errors,omitempty"`
	Synthesis     ReviewAgentSynthesis    `json:"synthesis"`
	StartedAt     string                  `json:"started_at"`
	CompletedAt   string                  `json:"completed_at"`
	GitLinkWrites int                     `json:"gitlink_writes"`
}

type ReviewAgentProvider interface {
	Assess(context.Context, ReviewAgentInvocation) (ReviewAgentAssessment, error)
}

type HTTPReviewAgentProvider struct {
	Endpoint   string
	Credential string
	HTTP       *http.Client
}

func NewHTTPReviewAgentProvider(endpoint, credentialReference string, httpClient *http.Client) (*HTTPReviewAgentProvider, error) {
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("Agent endpoint must be an absolute URL without userinfo or fragment")
	}
	if parsed.Scheme != "https" {
		host := parsed.Hostname()
		ip := net.ParseIP(host)
		if parsed.Scheme != "http" || (!strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback())) {
			return nil, fmt.Errorf("Agent endpoint must use HTTPS; HTTP is allowed only for loopback tests")
		}
	}
	credential := ""
	credentialReference = strings.TrimSpace(credentialReference)
	if credentialReference != "" {
		if !strings.HasPrefix(credentialReference, "env:") {
			return nil, fmt.Errorf("Agent credential reference must use env:VARIABLE")
		}
		name := strings.TrimPrefix(credentialReference, "env:")
		if name == "" || strings.ContainsAny(name, "= \t\r\n") {
			return nil, fmt.Errorf("Agent credential environment variable is invalid")
		}
		credential = os.Getenv(name)
		if strings.TrimSpace(credential) == "" {
			return nil, fmt.Errorf("Agent credential reference %q is not configured", credentialReference)
		}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	return &HTTPReviewAgentProvider{Endpoint: endpoint, Credential: credential, HTTP: httpClient}, nil
}

func (p *HTTPReviewAgentProvider) Assess(ctx context.Context, invocation ReviewAgentInvocation) (ReviewAgentAssessment, error) {
	if p == nil || p.HTTP == nil {
		return ReviewAgentAssessment{}, fmt.Errorf("Agent HTTP provider is not configured")
	}
	payload, err := json.Marshal(invocation)
	if err != nil {
		return ReviewAgentAssessment{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return ReviewAgentAssessment{}, err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Accept", "application/json")
	if p.Credential != "" {
		request.Header.Set("Authorization", "Bearer "+p.Credential)
	}
	response, err := p.HTTP.Do(request)
	if err != nil {
		return ReviewAgentAssessment{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil {
		return ReviewAgentAssessment{}, err
	}
	if len(body) > 1<<20 {
		return ReviewAgentAssessment{}, fmt.Errorf("Agent response exceeds 1 MiB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ReviewAgentAssessment{}, fmt.Errorf("Agent endpoint returned HTTP %d", response.StatusCode)
	}
	var assessment ReviewAgentAssessment
	if err := json.Unmarshal(body, &assessment); err != nil {
		return ReviewAgentAssessment{}, fmt.Errorf("decode Agent assessment: %w", err)
	}
	if assessment.RunID != invocation.RunID || assessment.TaskID != invocation.Task.TaskID ||
		assessment.Role != invocation.Task.Role || assessment.HeadSHA != invocation.HeadSHA {
		return ReviewAgentAssessment{}, fmt.Errorf("Agent assessment identity does not match invocation")
	}
	if validation := validateReviewAgentAssessment(assessment); validation != "" {
		return ReviewAgentAssessment{}, fmt.Errorf("invalid Agent assessment: %s", validation)
	}
	return assessment, nil
}

func RunReviewAgentPlan(
	ctx context.Context,
	plan ReviewAgentPlan,
	provider ReviewAgentProvider,
	taskTimeout time.Duration,
	now func() time.Time,
) ReviewAgentRun {
	if now == nil {
		now = time.Now
	}
	started := now().UTC()
	result := ReviewAgentRun{
		SchemaVersion: reviewAgentRunSchema,
		Status:        "running",
		RunID:         plan.RunID,
		Plan:          plan,
		Assessments:   []ReviewAgentAssessment{},
		StartedAt:     started.Format(time.RFC3339Nano),
		GitLinkWrites: 0,
	}
	if provider == nil || plan.SchemaVersion != reviewAgentPlanSchema || plan.GitLinkWrites != 0 {
		result.Status = "failed"
		result.Errors = append(result.Errors, "valid read-only Agent plan and provider are required")
		result.Synthesis = SynthesizeReviewAssessments(plan, nil)
		result.CompletedAt = now().UTC().Format(time.RFC3339Nano)
		return result
	}
	if taskTimeout <= 0 {
		taskTimeout = 2 * time.Minute
	}
	concurrency := plan.MaxConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 8 {
		concurrency = 8
	}
	type outcome struct {
		index      int
		assessment ReviewAgentAssessment
		err        error
	}
	semaphore := make(chan struct{}, concurrency)
	outcomes := make(chan outcome, len(plan.Tasks))
	var workers sync.WaitGroup
	for index, task := range plan.Tasks {
		workers.Add(1)
		go func(index int, task ReviewAgentTask) {
			defer workers.Done()
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				outcomes <- outcome{index: index, err: ctx.Err()}
				return
			}
			defer func() { <-semaphore }()
			taskCtx, cancel := context.WithTimeout(ctx, taskTimeout)
			defer cancel()
			assessment, err := provider.Assess(taskCtx, ReviewAgentInvocation{
				SchemaVersion:     "review.agent-invocation/v1",
				RunID:             plan.RunID,
				Repository:        plan.Repository,
				PRNumber:          plan.PRNumber,
				HeadSHA:           plan.HeadSHA,
				SourceFingerprint: plan.SourceFingerprint,
				Mode:              plan.Mode,
				Task:              task,
				GitLinkWrites:     0,
			})
			outcomes <- outcome{index: index, assessment: assessment, err: err}
		}(index, task)
	}
	workers.Wait()
	close(outcomes)
	ordered := make([]ReviewAgentAssessment, len(plan.Tasks))
	valid := make([]bool, len(plan.Tasks))
	for item := range outcomes {
		if item.err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", plan.Tasks[item.index].TaskID, item.err.Error()))
			continue
		}
		ordered[item.index] = item.assessment
		valid[item.index] = true
	}
	for index := range ordered {
		if valid[index] {
			result.Assessments = append(result.Assessments, ordered[index])
		}
	}
	result.Synthesis = SynthesizeReviewAssessments(plan, result.Assessments)
	if len(result.Errors) > 0 {
		result.Status = "incomplete"
	} else {
		result.Status = result.Synthesis.Status
	}
	result.CompletedAt = now().UTC().Format(time.RFC3339Nano)
	return result
}

func newReviewAgentRunShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-agent-run",
		Description: "Run a bounded read-only Review Agent plan through a provider-neutral HTTP contract",
		Flags: []common.Flag{
			{Name: "plan", Usage: "review.agent-plan/v1 JSON", Required: true},
			{Name: "endpoint", Usage: "HTTPS Agent assessment endpoint"},
			{Name: "credential-ref", Usage: "Optional Agent bearer credential as env:VARIABLE"},
			{Name: "task-timeout-seconds", Usage: "Timeout per specialist task", Default: "120"},
			{Name: "run", Usage: "Execute Agent requests. Without this flag, emit a no-network preview", Bool: true, Default: "false"},
		},
		Run: func(runtime *common.RuntimeContext) error {
			plan, err := readReviewAgentPlan(runtime.Arg("plan"))
			if err != nil {
				return err
			}
			if strings.ToLower(strings.TrimSpace(runtime.Arg("run"))) != "true" {
				return runtime.OutputData(map[string]interface{}{
					"schema_version": reviewAgentRunSchema,
					"status":         "preview",
					"plan":           plan,
					"network_calls":  0,
					"gitlink_writes": 0,
				})
			}
			provider, err := NewHTTPReviewAgentProvider(runtime.Arg("endpoint"), runtime.Arg("credential-ref"), nil)
			if err != nil {
				return err
			}
			seconds, err := strconv.Atoi(strings.TrimSpace(runtime.Arg("task-timeout-seconds")))
			if err != nil || seconds < 1 || seconds > 900 {
				return fmt.Errorf("--task-timeout-seconds must be between 1 and 900")
			}
			return runtime.OutputData(RunReviewAgentPlan(
				context.Background(),
				plan,
				provider,
				time.Duration(seconds)*time.Second,
				time.Now,
			))
		},
	}
}
