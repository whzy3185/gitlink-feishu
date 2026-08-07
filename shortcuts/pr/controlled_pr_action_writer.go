package pr

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

const (
	ControlledActionRejectClose = "reject_close"
	ControlledActionMerge       = "merge"
)

type ControlledPRActionOptions struct {
	Owner, Repository string
	PRNumber          int
	Action            string
	ExpectedHead      string
	ExpectedActor     string
	DryRun            bool
	BeforePOST        func() error
}

type ControlledPRActionResult struct {
	Status         string `json:"status"`
	Action         string `json:"action"`
	Repository     string `json:"repository"`
	PRNumber       int    `json:"pr_number"`
	CurrentHead    string `json:"current_head"`
	ExpectedHead   string `json:"expected_head"`
	Actor          string `json:"actor,omitempty"`
	RemoteState    string `json:"remote_state,omitempty"`
	Error          string `json:"error,omitempty"`
	MutationStatus string `json:"mutation_status"`
	POSTCount      int    `json:"post_count"`
	WouldWrite     bool   `json:"would_write"`
	Mutated        bool   `json:"mutated"`
}

// ExecuteControlledPRAction performs one guarded GitLink lifecycle mutation.
// The caller owns ActionPlan, lease, fingerprint, and reconciliation storage.
func ExecuteControlledPRAction(runtime *common.RuntimeContext, opts ControlledPRActionOptions) ControlledPRActionResult {
	action := strings.ToLower(strings.TrimSpace(opts.Action))
	result := ControlledPRActionResult{
		Action: action, Repository: opts.Owner + "/" + opts.Repository, PRNumber: opts.PRNumber,
		ExpectedHead: strings.TrimSpace(opts.ExpectedHead), MutationStatus: "none",
	}
	if runtime == nil || runtime.Client == nil || opts.Owner == "" || opts.Repository == "" || opts.PRNumber <= 0 {
		return controlledPRActionFailure(result, "controlled PR action runtime and target are required")
	}
	if action != ControlledActionRejectClose && action != ControlledActionMerge {
		return controlledPRActionFailure(result, "controlled PR action must be reject_close or merge")
	}
	runtimeCopy := *runtime
	clientCopy := *runtime.Client
	if clientCopy.HTTP == nil {
		clientCopy.HTTP = &http.Client{Timeout: 3 * time.Second}
	} else if clientCopy.HTTP.Timeout == 0 || clientCopy.HTTP.Timeout > 3*time.Second {
		httpCopy := *clientCopy.HTTP
		httpCopy.Timeout, clientCopy.HTTP = 3*time.Second, &httpCopy
	}
	runtimeCopy.Client, runtimeCopy.Owner, runtimeCopy.Repo = &clientCopy, opts.Owner, opts.Repository

	current, err := fetchControlledPRContext(&runtimeCopy, opts.PRNumber)
	if err != nil {
		return controlledPRActionFailure(result, "current pull request is required")
	}
	result.CurrentHead = strings.TrimSpace(current.CurrentHeadSHA)
	result.RemoteState = strings.ToLower(strings.TrimSpace(current.WorkItem.GitLinkState))
	if result.RemoteState != "open" || result.CurrentHead == "" {
		return controlledPRActionFailure(result, "pull request must be open with a current head")
	}
	if result.ExpectedHead == "" {
		result.ExpectedHead = result.CurrentHead
	}
	if result.ExpectedHead != result.CurrentHead {
		result.Status, result.Error = "stale", "expected head does not match current head"
		return result
	}
	if action == ControlledActionMerge && controlledPRCIHasFailed(current.WorkItem.CISummary.State) {
		return controlledPRActionFailure(result, "merge is blocked by an explicitly failed CI state")
	}

	user, err := runtimeCopy.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return controlledPRActionFailure(result, "resolve active GitLink identity: "+err.Error())
	}
	userMap, _ := user.Data.(map[string]interface{})
	result.Actor = commonReviewString(userMap, "login", "username")
	if result.Actor == "" {
		return controlledPRActionFailure(result, "active GitLink identity is missing login")
	}
	if opts.ExpectedActor != "" && result.Actor != strings.TrimSpace(opts.ExpectedActor) {
		return controlledPRActionFailure(result, "active GitLink identity does not match the expected actor")
	}
	if opts.DryRun {
		result.Status, result.WouldWrite = "dry_run", true
		return result
	}
	if opts.BeforePOST != nil {
		if err := opts.BeforePOST(); err != nil {
			return controlledPRActionFailure(result, "persist remote write boundary: "+err.Error())
		}
	}

	result.MutationStatus = "possible"
	path, payload := fmt.Sprintf("/%s/%s/pulls/%d/refuse_merge", opts.Owner, opts.Repository, opts.PRNumber), map[string]interface{}(nil)
	if action == ControlledActionMerge {
		path = fmt.Sprintf("/%s/%s/pulls/%d/pr_merge", opts.Owner, opts.Repository, opts.PRNumber)
		payload = map[string]interface{}{"do": "merge"}
	}
	_, postErr := runtimeCopy.CallAPI("POST", path, payload)
	result.POSTCount = 1
	if postErr == nil {
		result.Mutated = true
	}
	for attempt := 0; attempt < 3; attempt++ {
		observed, readErr := fetchControlledPRContext(&runtimeCopy, opts.PRNumber)
		if readErr == nil {
			result.CurrentHead = strings.TrimSpace(observed.CurrentHeadSHA)
			result.RemoteState = strings.ToLower(strings.TrimSpace(observed.WorkItem.GitLinkState))
			if controlledPRActionVerified(action, result.RemoteState) {
				result.Status, result.Mutated, result.MutationStatus = "verified", true, "confirmed"
				return result
			}
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
		}
	}
	result.Status = "unknown"
	if postErr != nil {
		result.Error = postErr.Error()
	} else {
		result.Error = "GitLink mutation succeeded but bounded GET read-back did not verify it"
	}
	return result
}

func fetchControlledPRContext(runtime *common.RuntimeContext, number int) (workflow.ReviewContext, error) {
	return workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner: runtime.Owner, Repo: runtime.Repo, Number: number, IncludePR: true, IncludeVersions: true,
	})
}

func controlledPRActionFailure(result ControlledPRActionResult, message string) ControlledPRActionResult {
	result.Status, result.Error, result.MutationStatus = "failed", message, "none"
	return result
}

func controlledPRActionVerified(action, state string) bool {
	return action == ControlledActionMerge && state == "merged" || action == ControlledActionRejectClose && state == "closed"
}

func controlledPRCIHasFailed(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "failed", "failure", "error":
		return true
	default:
		return false
	}
}
