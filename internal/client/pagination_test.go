package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestPaginateAllKeyResourceWrappedPages(t *testing.T) {
	// GitLink v1 list shape: {"total_count":N, "issues":[...]}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit != 2 {
			t.Fatalf("limit = %d, want 2", limit)
		}
		var items []map[string]interface{}
		switch page {
		case 1:
			items = []map[string]interface{}{{"id": 1}, {"id": 2}}
		case 2:
			items = []map[string]interface{}{{"id": 3}}
		default:
			t.Fatalf("unexpected page %d", page)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 3,
			"issues":      items,
		})
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "2")
	items, err := c.PaginateAllKey("/owner/repo/issues", params, "issues")
	if err != nil {
		t.Fatalf("PaginateAllKey: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
}

func TestPaginateAllKeyStopsAtTotalCount(t *testing.T) {
	// A broken endpoint that keeps returning full pages must stop at total_count.
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 4,
			"pulls":       []map[string]interface{}{{"id": 1}, {"id": 2}},
		})
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "2")
	items, err := c.PaginateAllKey("/owner/repo/pulls", params, "pulls")
	if err != nil {
		t.Fatalf("PaginateAllKey: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(items))
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestPaginateAllAutoDetectsUniqueArrayField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"branches":    []map[string]interface{}{{"name": "master"}},
		})
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/owner/repo/branches", nil)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestPaginateAllDataWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":1}],"total_count":1}`)
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/things", nil)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestPaginateAllSingleObjectFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":42,"name":"solo"}`)
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/thing", nil)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(items[0], &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if obj["name"] != "solo" {
		t.Fatalf("name = %v, want solo", obj["name"])
	}
}

func TestPaginateAllKeyMissingKeyNotList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":42}`)
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAllKey("/thing", nil, "issues")
	if err != nil {
		t.Fatalf("PaginateAllKey: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1 (single-object fallback)", len(items))
	}
}
