package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestPaginateAllKeyConcurrentPagesOrdered(t *testing.T) {
	// With total_count known after page 1, pages 2..N are fetched
	// concurrently; the combined result must stay in page order.
	const total = 25
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		var items []map[string]interface{}
		for i := (page-1)*limit + 1; i <= page*limit && i <= total; i++ {
			items = append(items, map[string]interface{}{"id": i})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": total,
			"issues":      items,
		})
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "4")
	items, err := c.PaginateAllKey("/repos/o/r/issues", params, "issues")
	if err != nil {
		t.Fatalf("PaginateAllKey: %v", err)
	}
	if len(items) != total {
		t.Fatalf("len = %d, want %d", len(items), total)
	}
	for i, raw := range items {
		var obj struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			t.Fatalf("unmarshal item %d: %v", i, err)
		}
		if obj.ID != i+1 {
			t.Fatalf("item %d id = %d, want %d (page order broken)", i, obj.ID, i+1)
		}
	}
}

func TestPaginateAllKeyConcurrentPageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 10,
			"issues":      []map[string]interface{}{{"id": page*2 - 1}, {"id": page * 2}},
		})
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "2")
	if _, err := c.PaginateAllKey("/repos/o/r/issues", params, "issues"); err == nil {
		t.Fatal("expected error from failing page")
	}
}

func TestPaginateAllKeyServerCappedLimit(t *testing.T) {
	// The server caps every page at 2 items regardless of the requested
	// limit; with total_count reported, all 5 items must still be fetched.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		start := (page - 1) * 2
		var items []string
		for i := start; i < start+2 && i < 5; i++ {
			items = append(items, fmt.Sprintf(`{"id":%d}`, i))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"count":5,"users":[%s]}`, strings.Join(items, ","))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "100")
	items, err := c.PaginateAllKey("/thing", params, "users")
	if err != nil {
		t.Fatalf("PaginateAllKey: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}
}

// BenchmarkPaginateAllKey measures full-list pagination against a server with
// simulated per-page latency, exercising the concurrent page-fetch path.
func BenchmarkPaginateAllKey(b *testing.B) {
	const total, perPage = 500, 50
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		var items []string
		for i := (page-1)*perPage + 1; i <= page*perPage && i <= total; i++ {
			items = append(items, fmt.Sprintf(`{"id":%d}`, i))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"total_count":%d,"issues":[%s]}`, total, strings.Join(items, ","))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params := url.Values{}
		params.Set("limit", strconv.Itoa(perPage))
		items, err := c.PaginateAllKey("/repos/o/r/issues", params, "issues")
		if err != nil {
			b.Fatalf("PaginateAllKey: %v", err)
		}
		if len(items) != total {
			b.Fatalf("len = %d, want %d", len(items), total)
		}
	}
}
