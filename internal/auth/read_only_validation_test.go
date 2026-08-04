package auth

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type readOnlyValidationRoundTripFunc func(*http.Request) (*http.Response, error)

func (f readOnlyValidationRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestReadOnlyValidationTransportPersistsCountsAndBlocksWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "methods.json")
	called := 0
	transport := &gitLinkReadOnlyValidationTransport{
		logPath: path,
		base: readOnlyValidationRoundTripFunc(func(*http.Request) (*http.Response, error) {
			called++
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}),
	}
	request, _ := http.NewRequest(http.MethodGet, "https://gitlink.example/api/repository", nil)
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	request, _ = http.NewRequest(http.MethodDelete, "https://gitlink.example/api/repository", nil)
	if _, err := transport.RoundTrip(request); err == nil {
		t.Fatal("expected write to be blocked")
	}
	if called != 1 {
		t.Fatalf("network call count = %d", called)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"GET": 1`) || !strings.Contains(text, `"DELETE": 1`) {
		t.Fatalf("method log = %s", text)
	}
}
