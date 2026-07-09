package capability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
)

// Status represents the availability status of a backend API domain.
type Status int

const (
	StatusUnknown     Status = iota // not probed yet
	StatusAvailable                 // backend API responds with JSON
	StatusUnavailable               // backend API returns HTML or 404
	StatusError                     // probe itself failed (network error, etc.)
)

func (s Status) String() string {
	switch s {
	case StatusAvailable:
		return "available"
	case StatusUnavailable:
		return "unavailable"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Emoji returns a single-character status indicator for help text.
func (s Status) Emoji() string {
	switch s {
	case StatusAvailable:
		return "✓"
	case StatusUnavailable:
		return "⚠"
	case StatusError:
		return "✗"
	default:
		return "?"
	}
}

// DomainStatus records the probe result for a single domain.
type DomainStatus struct {
	Domain      string    `json:"domain"`
	Status      Status    `json:"status"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

// CanaryProbe defines a lightweight endpoint used to test domain availability.
type CanaryProbe struct {
	Method    string // HTTP method (usually GET)
	Path      string // API path; use {owner} and {repo} as placeholders
	NeedsRepo bool   // whether the probe requires owner/repo context
}

// canaryEndpoints maps each domain to its probe endpoint.
var canaryEndpoints = map[string]CanaryProbe{
	"label":        {Method: "GET", Path: "/v1/{owner}/{repo}/issue_tags", NeedsRepo: true},
	"notification": {Method: "GET", Path: "/notifications?page=1&limit=1", NeedsRepo: false},
	"pm":           {Method: "GET", Path: "/pm/dashboards", NeedsRepo: false},
	"wiki":         {Method: "GET", Path: "/{owner}/{repo}/wiki_pages", NeedsRepo: true},
	"pipeline":     {Method: "GET", Path: "/pm/pipelines", NeedsRepo: false},
	"webhook":      {Method: "GET", Path: "/v1/{owner}/{repo}/webhooks", NeedsRepo: true},
	"member":       {Method: "GET", Path: "/{owner}/{repo}/collaborators", NeedsRepo: true},
	"milestone":    {Method: "GET", Path: "/v1/{owner}/{repo}/milestones", NeedsRepo: true},
	"export":       {Method: "GET", Path: "/{owner}/{repo}/contributors", NeedsRepo: true},
	"search":       {Method: "GET", Path: "/repos/search?q=test&limit=1", NeedsRepo: false},
	"workflow":     {Method: "GET", Path: "/v1/{owner}/{repo}", NeedsRepo: true},
}

// Registry holds capability probe results with thread-safe access.
type Registry struct {
	mu        sync.RWMutex
	statuses  map[string]*DomainStatus
	cachePath string
}

// NewRegistry creates a Registry and attempts to load cached results.
func NewRegistry() *Registry {
	r := &Registry{
		statuses:  make(map[string]*DomainStatus),
		cachePath: cacheFilePath(),
	}
	r.load()
	return r
}

// Get returns the cached status for a domain.
func (r *Registry) Get(domain string) Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ds, ok := r.statuses[domain]; ok {
		return ds.Status
	}
	return StatusUnknown
}

// GetAll returns a copy of all domain statuses.
func (r *Registry) GetAll() map[string]*DomainStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*DomainStatus, len(r.statuses))
	for k, v := range r.statuses {
		copy := *v
		result[k] = &copy
	}
	return result
}

// ProbeAll probes all registered domains concurrently.
// owner and repo are used for endpoints that require repository context.
// If owner/repo are empty, repo-dependent probes are skipped.
func (r *Registry) ProbeAll(cli *client.Client, owner, repo string) map[string]*DomainStatus {
	results := make(map[string]*DomainStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for domain, canary := range canaryEndpoints {
		if canary.NeedsRepo && (owner == "" || repo == "") {
			// Skip repo-dependent probes when no repo context available
			continue
		}
		wg.Add(1)
		go func(domain string, canary CanaryProbe) {
			defer wg.Done()
			ds := r.probeOne(cli, domain, canary, owner, repo)
			mu.Lock()
			results[domain] = ds
			mu.Unlock()
		}(domain, canary)
	}
	wg.Wait()

	// Merge results into registry
	r.mu.Lock()
	for k, v := range results {
		r.statuses[k] = v
	}
	r.mu.Unlock()

	r.save()
	return results
}

// probeOne probes a single canary endpoint.
func (r *Registry) probeOne(cli *client.Client, domain string, canary CanaryProbe, owner, repo string) *DomainStatus {
	path := canary.Path
	if canary.NeedsRepo {
		path = strings.Replace(path, "{owner}", owner, 1)
		path = strings.Replace(path, "{repo}", repo, 1)
	}

	ds := &DomainStatus{
		Domain:      domain,
		LastChecked: time.Now(),
	}

	env, err := cli.Do(canary.Method, path, nil, nil)
	if err != nil {
		apiErr, ok := err.(*client.APIError)
		if !ok {
			ds.Status = StatusError
			ds.Message = fmt.Sprintf("网络错误: %v", err)
			return ds
		}

		switch {
		case apiErr.Code == "HTML_RESPONSE":
			// Backend returned HTML instead of JSON — endpoint doesn't exist
			ds.Status = StatusUnavailable
			ds.Message = "后端 API 尚未实现该端点"

		case apiErr.StatusCode == 404:
			// 404 means the endpoint path doesn't exist on the backend
			ds.Status = StatusUnavailable
			ds.Message = "API 端点不存在（404）"

		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			// Auth/permission errors mean the endpoint EXISTS but probe lacks credentials.
			// The user may have valid credentials — mark as available.
			ds.Status = StatusAvailable
			ds.Message = "端点存在（探测权限受限，用户可能有完整权限）"

		default:
			// Other HTTP errors (422, 500, etc.) — endpoint exists but something went wrong
			ds.Status = StatusAvailable
			ds.Message = fmt.Sprintf("端点响应: HTTP %d", apiErr.StatusCode)
		}
		return ds
	}

	if env != nil && env.OK {
		ds.Status = StatusAvailable
		ds.Message = "API 正常响应"
	} else {
		ds.Status = StatusUnavailable
		if env != nil && env.Error != nil {
			ds.Message = env.Error.Message
		}
	}
	return ds
}

// Refresh re-probes all domains and returns the updated statuses.
func (r *Registry) Refresh(cli *client.Client, owner, repo string) map[string]*DomainStatus {
	return r.ProbeAll(cli, owner, repo)
}

// IsStale returns true if the cache is older than 24 hours or doesn't exist.
func (r *Registry) IsStale() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ds := range r.statuses {
		if time.Since(ds.LastChecked) > 24*time.Hour {
			return true
		}
	}
	return len(r.statuses) == 0
}

// Summary returns a human-readable multi-line summary of all domain statuses.
func (r *Registry) Summary() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("API 后端能力探测结果:\n")
	sb.WriteString(strings.Repeat("-", 50) + "\n")

	// Order domains for consistent output
	domains := []string{
		"label", "notification", "pm", "wiki", "pipeline",
		"webhook", "member", "milestone", "export", "search", "workflow",
	}
	for _, domain := range domains {
		ds, ok := r.statuses[domain]
		if !ok {
			sb.WriteString(fmt.Sprintf("  ? %-15s 未探测\n", domain))
			continue
		}
		icon := ds.Status.Emoji()
		statusText := ds.Status.String()
		detail := ""
		if ds.Message != "" {
			detail = " — " + ds.Message
		}
		sb.WriteString(fmt.Sprintf("  %s %-15s %s%s\n", icon, domain, statusText, detail))
	}
	return sb.String()
}

// cacheFilePath returns the path to the capability cache file.
func cacheFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "gitlink-cli", "capabilities.json")
}

// save writes the current registry state to the cache file.
func (r *Registry) save() {
	r.mu.RLock()
	data, err := json.MarshalIndent(r.statuses, "", "  ")
	r.mu.RUnlock()
	if err != nil {
		return
	}
	dir := filepath.Dir(r.cachePath)
	os.MkdirAll(dir, 0700)
	os.WriteFile(r.cachePath, data, 0600)
}

// load reads cached capability data from disk.
func (r *Registry) load() {
	data, err := os.ReadFile(r.cachePath)
	if err != nil {
		return
	}
	var statuses map[string]*DomainStatus
	if err := json.Unmarshal(data, &statuses); err != nil {
		return
	}
	r.mu.Lock()
	r.statuses = statuses
	r.mu.Unlock()
}
