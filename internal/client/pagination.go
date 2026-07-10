package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
)

// maxPaginationPages caps auto-pagination as a safety guard against
// endpoints that ignore the page parameter and keep returning data.
const maxPaginationPages = 1000

// paginationWorkers bounds concurrent page fetches when the total page
// count is known after the first page, so remaining pages can be fetched
// in parallel without overwhelming the server.
const paginationWorkers = 5

// PaginateAll fetches all pages and returns combined results.
// The list array is auto-detected inside the response body.
func (c *Client) PaginateAll(path string, params url.Values) ([]json.RawMessage, error) {
	return c.PaginateAllKey(path, params, "")
}

// PaginateAllKey fetches all pages, extracting the list array from the
// response field named listKey (e.g. "issues", "pulls", "branches").
// When listKey is empty the array is auto-detected: top-level arrays,
// the conventional "data" wrapper, or a unique array-valued field.
// Pagination stops when a page returns fewer items than the limit, when
// total_count (if reported) is reached, or at the safety page cap.
func (c *Client) PaginateAllKey(path string, params url.Values, listKey string) ([]json.RawMessage, error) {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("limit") == "" {
		params.Set("limit", "50")
	}

	var all []json.RawMessage
	totalCount := -1

	for page := 1; page <= maxPaginationPages; page++ {
		if page == 2 && totalCount >= 0 {
			perPage := len(all)
			if perPage > 0 && totalCount > perPage {
				rest, err := c.fetchPagesConcurrent(path, params, listKey, perPage, totalCount)
				if err != nil {
					return nil, err
				}
				all = append(all, rest...)
			}
			break
		}
		params.Set("page", strconv.Itoa(page))
		env, err := c.Get(path, params)
		if err != nil {
			return nil, err
		}
		if !env.OK {
			return nil, fmt.Errorf("API error on page %d", page)
		}

		items, pageTotal, isList := extractListItems(env.Data, listKey)
		if !isList {
			if page == 1 {
				raw, _ := json.Marshal(env.Data)
				return []json.RawMessage{raw}, nil
			}
			break
		}
		if pageTotal >= 0 {
			totalCount = pageTotal
		}

		if len(items) == 0 {
			break
		}
		all = append(all, items...)

		if totalCount >= 0 && len(all) >= totalCount {
			break
		}
		// A short page only signals the end when the endpoint does not
		// report a total: servers may cap the requested limit (e.g. ask
		// for 100, get 20 per page), so with a known total we rely on it.
		limit, _ := strconv.Atoi(params.Get("limit"))
		if totalCount < 0 && len(items) < limit {
			break
		}
	}

	return all, nil
}

// fetchPagesConcurrent fetches pages 2..N in parallel with a bounded worker
// pool, preserving page order in the returned slice. It is only used when
// the endpoint reported a total_count, so the page count is known upfront.
func (c *Client) fetchPagesConcurrent(path string, params url.Values, listKey string, perPage, totalCount int) ([]json.RawMessage, error) {
	lastPage := (totalCount + perPage - 1) / perPage
	if lastPage > maxPaginationPages {
		lastPage = maxPaginationPages
	}

	type pageResult struct {
		items []json.RawMessage
		err   error
	}
	results := make([]pageResult, lastPage+1)

	var wg sync.WaitGroup
	sem := make(chan struct{}, paginationWorkers)
	for page := 2; page <= lastPage; page++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			q := url.Values{}
			for k, vs := range params {
				q[k] = append([]string(nil), vs...)
			}
			q.Set("page", strconv.Itoa(page))
			env, err := c.Get(path, q)
			if err != nil {
				results[page] = pageResult{err: err}
				return
			}
			if !env.OK {
				results[page] = pageResult{err: fmt.Errorf("API error on page %d", page)}
				return
			}
			items, _, isList := extractListItems(env.Data, listKey)
			if !isList {
				return
			}
			results[page] = pageResult{items: items}
		}(page)
	}
	wg.Wait()

	var all []json.RawMessage
	for page := 2; page <= lastPage; page++ {
		if results[page].err != nil {
			return nil, results[page].err
		}
		all = append(all, results[page].items...)
	}
	if remaining := totalCount - perPage; len(all) > remaining {
		all = all[:remaining]
	}
	return all, nil
}

// extractListItems locates the list array inside a decoded response body.
// It returns the items, the reported total_count (-1 when absent) and
// whether a list array was found at all.
func extractListItems(data interface{}, listKey string) ([]json.RawMessage, int, bool) {
	switch v := data.(type) {
	case []interface{}:
		return marshalItems(v), -1, true
	case map[string]interface{}:
		total := -1
		if tc, ok := v["total_count"].(float64); ok {
			total = int(tc)
		} else if tc, ok := v["count"].(float64); ok {
			// Some endpoints (e.g. /users/:login/projects) report "count".
			total = int(tc)
		}
		if listKey != "" {
			if slice, ok := v[listKey].([]interface{}); ok {
				return marshalItems(slice), total, true
			}
			return nil, total, false
		}
		if slice, ok := v["data"].([]interface{}); ok {
			return marshalItems(slice), total, true
		}
		// Auto-detect: GitLink v1 list endpoints wrap the array in a
		// resource-named field ({"total_count":N,"issues":[...]}).
		var found []interface{}
		arrays := 0
		for _, val := range v {
			if slice, ok := val.([]interface{}); ok {
				arrays++
				found = slice
			}
		}
		if arrays == 1 {
			return marshalItems(found), total, true
		}
		return nil, total, false
	}
	return nil, -1, false
}

func marshalItems(items []interface{}) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		raw, _ := json.Marshal(item)
		out = append(out, raw)
	}
	return out
}
