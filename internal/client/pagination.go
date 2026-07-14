package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// PaginateAll fetches all pages and returns combined results.
func (c *Client) PaginateAll(path string, params url.Values) ([]json.RawMessage, error) {
	if params == nil {
		params = url.Values{}
	}
	if params.Get("limit") == "" {
		params.Set("limit", "50")
	}

	var all []json.RawMessage
	page := 1

	for {
		params.Set("page", strconv.Itoa(page))
		env, err := c.Get(path, params)
		if err != nil {
			return nil, err
		}

		if !env.OK {
			return nil, fmt.Errorf("API error on page %d", page)
		}

		// Try to extract array from data
		var items []json.RawMessage
		switch data := env.Data.(type) {
		case []interface{}:
			for _, item := range data {
				raw, _ := json.Marshal(item)
				items = append(items, raw)
			}
		case map[string]interface{}:
			arr, ok := listArray(data)
			if !ok {
				// Single object, not paginated
				raw, _ := json.Marshal(data)
				return []json.RawMessage{raw}, nil
			}
			for _, item := range arr {
				raw, _ := json.Marshal(item)
				items = append(items, raw)
			}
		}

		if len(items) == 0 {
			break
		}

		all = append(all, items...)

		// Check if we got fewer items than limit
		limit, _ := strconv.Atoi(params.Get("limit"))
		if len(items) < limit {
			break
		}

		page++
	}

	return all, nil
}

// listArray resolves the item array of one wrapped list page. GitLink wraps the
// array under the generic "data" key on some endpoints and under a
// resource-specific key on others ({"pulls":[...]}, {"issues":[...]},
// {"branches":[...]}, ...), so prefer "data" and otherwise accept the sole
// array-valued field. A map with no array field — or several, which is
// ambiguous — is not a list page, so ok is false.
func listArray(data map[string]interface{}) (arr []interface{}, ok bool) {
	if d, isArr := data["data"].([]interface{}); isArr {
		return d, true
	}
	for _, v := range data {
		slice, isArr := v.([]interface{})
		if !isArr {
			continue
		}
		if ok {
			return nil, false
		}
		arr, ok = slice, true
	}
	return arr, ok
}
