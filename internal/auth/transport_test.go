package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestTransportCookieAuth(t *testing.T) {
	// Mock an HTTP server that checks for the Cookie header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := r.Header.Get("Cookie")
		if cookie == "" {
			t.Error("expected Cookie header")
		}

		// Verify the request has the right Accept header
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}

		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	// Set a cookie-based token
	os.Setenv("GITLINK_TOKEN", "cookie:autologin_trustie=test123")
	defer os.Unsetenv("GITLINK_TOKEN")

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTransportTokenAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") == "" {
			t.Error("expected access_token query parameter")
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}

		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	os.Setenv("GITLINK_TOKEN", "private-token-abc")
	defer os.Unsetenv("GITLINK_TOKEN")

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTransportCookieAppend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := r.Header.Get("Cookie")
		if cookie == "" {
			t.Error("expected Cookie header")
		}
		// Should contain both original and injected cookies
		if cookie != "existing=val; autologin_trustie=injected" {
			t.Errorf("Cookie = %q, want 'existing=val; autologin_trustie=injected'", cookie)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	os.Setenv("GITLINK_TOKEN", "cookie:autologin_trustie=injected")
	defer os.Unsetenv("GITLINK_TOKEN")

	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Cookie", "existing=val")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestTransportNoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	// Force empty env and redirect HOME to avoid keychain fallback
	os.Unsetenv("GITLINK_TOKEN")
	oldHome := os.Getenv("HOME")
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", oldHome)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestTransportDefaultContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	// Transport only sets Content-Type when Body is non-nil
	body := strings.NewReader(`{"key":"val"}`)
	req, _ := http.NewRequest("POST", server.URL, body)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestTransportExplicitContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should preserve explicit Content-Type
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("POST", server.URL, nil)
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Transport == nil {
		t.Fatal("expected Transport to be set")
	}
	if _, ok := client.Transport.(*Transport); !ok {
		t.Fatalf("expected *Transport, got %T", client.Transport)
	}
}

func TestTransportEnvVarPriority(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "env-token" {
			t.Errorf("access_token = %q, want env-token", r.URL.Query().Get("access_token"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	os.Setenv("GITLINK_TOKEN", "env-token")
	defer os.Unsetenv("GITLINK_TOKEN")

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestTransportNilBase(t *testing.T) {
	// When Base is nil, it should use http.DefaultTransport
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	transport := &Transport{Base: nil}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}
