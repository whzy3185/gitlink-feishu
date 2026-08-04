package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const gitLinkReadOnlyValidationLogEnvironment = "GITLINK_REVIEW_READ_ONLY_METHOD_LOG"

var gitLinkReadOnlyValidationMu sync.Mutex

type gitLinkReadOnlyValidationTransport struct {
	base    http.RoundTripper
	logPath string
}

func (t *gitLinkReadOnlyValidationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil {
		return nil, fmt.Errorf("GitLink validation request is required")
	}
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if err := recordGitLinkValidationMethod(t.logPath, method); err != nil {
		return nil, fmt.Errorf("record GitLink validation method: %w", err)
	}
	if method != http.MethodGet && method != http.MethodHead {
		return nil, fmt.Errorf("GitLink validation blocked %s before network transmission", method)
	}
	return t.base.RoundTrip(request)
}

func wrapGitLinkReadOnlyValidation(base http.RoundTripper) http.RoundTripper {
	logPath := strings.TrimSpace(os.Getenv(gitLinkReadOnlyValidationLogEnvironment))
	if logPath == "" {
		return base
	}
	return &gitLinkReadOnlyValidationTransport{base: base, logPath: logPath}
}

func recordGitLinkValidationMethod(path, method string) error {
	gitLinkReadOnlyValidationMu.Lock()
	defer gitLinkReadOnlyValidationMu.Unlock()
	counts := map[string]int{"GET": 0, "HEAD": 0, "POST": 0, "PUT": 0, "PATCH": 0, "DELETE": 0}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &counts)
	} else if !os.IsNotExist(err) {
		return err
	}
	counts[method]++
	data, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
