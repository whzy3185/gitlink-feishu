package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/auth"
	"github.com/gitlink-org/gitlink-cli/internal/config"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

type Client struct {
	HTTP    *http.Client
	BaseURL string
	Debug   bool
}

type APIError struct {
	StatusCode int
	Code       interface{}
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%v] %s", e.Code, e.Message)
}

func New() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &Client{
		HTTP:    auth.NewHTTPClient(),
		BaseURL: cfg.BaseURL,
	}, nil
}

// Do makes an API call with automatic .json suffix appended.
func (c *Client) Do(method, path string, body interface{}, query url.Values) (*output.Envelope, error) {
	path = normalizeAPIPath(c.BaseURL, path)
	return c.do(method, path, body, query, true, "json")
}

// DoRaw makes an API call without appending .json suffix.
func (c *Client) DoRaw(method, path string, body interface{}, query url.Values) (*output.Envelope, error) {
	return c.do(method, path, body, query, false, "json")
}

// DoForm makes an API call with form-encoded body (no .json suffix).
// Used for Wiki and other endpoints that expect application/x-www-form-urlencoded.
func (c *Client) DoForm(method, path string, body url.Values, query url.Values) (*output.Envelope, error) {
	return c.do(method, path, body, query, false, "form")
}

func (c *Client) do(method, path string, body interface{}, query url.Values, appendJSON bool, encoding string) (*output.Envelope, error) {
	if appendJSON {
		if idx := strings.Index(path, "?"); idx != -1 {
			basePath := path[:idx]
			queryStr := path[idx:]
			if shouldAppendJSONSuffix(basePath) {
				path = basePath + ".json" + queryStr
			}
		} else if shouldAppendJSONSuffix(path) {
			path += ".json"
		}
	}
	fullURL := c.BaseURL + path
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		fullURL += sep + query.Encode()
	}

	var bodyData []byte
	var bodyReader io.Reader
	var contentType string
	if body != nil {
		if encoding == "form" {
			formValues, ok := body.(url.Values)
			if !ok {
				return nil, fmt.Errorf("DoForm requires url.Values body")
			}
			bodyData = []byte(formValues.Encode())
			contentType = "application/x-www-form-urlencoded"
		} else {
			var err error
			bodyData, err = json.Marshal(body)
			if err != nil {
				return nil, err
			}
			contentType = "application/json"
		}
		bodyReader = bytes.NewReader(bodyData)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.Debug {
		fmt.Printf("→ %s %s\n", method, fullURL)
		if bodyData != nil {
			fmt.Printf("  body: %s\n", string(bodyData))
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.Debug {
		fmt.Printf("← %d %s\n", resp.StatusCode, string(respData[:min(len(respData), 200)]))
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData))),
		}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respData, &raw); err != nil {
		return output.SuccessEnvelope(string(respData), nil), nil
	}

	// Check GitLink error-in-body pattern
	// Support both {"status":N, "message":"..."} and gateway {"code":N, "msg":"..."}
	var bodyCode float64
	var bodyMsg string
	if status, ok := raw["status"]; ok {
		switch v := status.(type) {
		case float64:
			bodyCode = v
		case int:
			bodyCode = float64(v)
		}
		bodyMsg, _ = raw["message"].(string)
	} else if code, ok := raw["code"]; ok {
		switch v := code.(type) {
		case float64:
			bodyCode = v
		case int:
			bodyCode = float64(v)
		}
		bodyMsg, _ = raw["msg"].(string)
		if bodyMsg == "" {
			bodyMsg, _ = raw["message"].(string)
		}
	}
	if bodyCode != 0 && bodyCode != 200 && bodyCode != 201 && bodyCode != 204 && bodyCode != 1 {
		suggestion := suggestFix(int(bodyCode))
		return output.ErrorEnvelope(int(bodyCode), bodyMsg, suggestion), &APIError{
			StatusCode: int(bodyCode),
			Code:       int(bodyCode),
			Message:    bodyMsg,
		}
	}

	if dataStr, ok := raw["data"].(string); ok {
		var parsedData interface{}
		if err := json.Unmarshal([]byte(dataStr), &parsedData); err == nil {
			raw["data"] = json.RawMessage(dataStr)
		}
	}

	var meta *output.Meta
	if tc, ok := raw["total_count"]; ok {
		meta = &output.Meta{}
		if v, ok := tc.(float64); ok {
			meta.TotalCount = int(v)
		}
		if v, ok := raw["page"].(float64); ok {
			meta.Page = int(v)
		}
		if v, ok := raw["limit"].(float64); ok {
			meta.Limit = int(v)
		}
	}

	return output.SuccessEnvelope(raw, meta), nil
}

func shouldAppendJSONSuffix(path string) bool {
	if strings.HasSuffix(path, ".json") {
		return false
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "raw" && i >= 2 && i+2 < len(parts) {
			return false
		}
	}
	// Wiki open API endpoints do not use .json suffix
	if len(parts) >= 3 && parts[0] == "wiki" && parts[1] == "open" {
		return false
	}
	return true
}

func normalizeAPIPath(baseURL, path string) string {
	if strings.HasSuffix(strings.TrimRight(baseURL, "/"), "/api") {
		switch {
		case path == "/api":
			return ""
		case strings.HasPrefix(path, "/api/"):
			return strings.TrimPrefix(path, "/api")
		}
	}
	return path
}

// DoRaw makes an API call without appending .json to the path.
func (c *Client) DoRaw(method, path string, body interface{}, query url.Values) (*output.Envelope, error) {
	fullURL := c.BaseURL + path
	if query != nil && len(query) > 0 {
		sep := "?"
		if strings.Contains(fullURL, "?") {
			sep = "&"
		}
		fullURL += sep + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Printf("→ %s %s\n", method, fullURL)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if c.Debug {
		fmt.Printf("← %d %s\n", resp.StatusCode, string(respData[:min(len(respData), 200)]))
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData))),
		}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respData, &raw); err != nil {
		return output.SuccessEnvelope(string(respData), nil), nil
	}

	if status, ok := raw["status"]; ok {
		var statusCode float64
		switch v := status.(type) {
		case float64:
			statusCode = v
		case int:
			statusCode = float64(v)
		}
		if statusCode != 0 && statusCode != 200 && statusCode != 1 {
			msg, _ := raw["message"].(string)
			suggestion := suggestFix(int(statusCode))
			return output.ErrorEnvelope(int(statusCode), msg, suggestion), &APIError{
				StatusCode: int(statusCode),
				Code:       int(statusCode),
				Message:    msg,
			}
		}
	}

	if dataStr, ok := raw["data"].(string); ok {
		var parsedData interface{}
		if err := json.Unmarshal([]byte(dataStr), &parsedData); err == nil {
			raw["data"] = json.RawMessage(dataStr)
		}
	}

	var meta *output.Meta
	if tc, ok := raw["total_count"]; ok {
		meta = &output.Meta{}
		if v, ok := tc.(float64); ok {
			meta.TotalCount = int(v)
		}
		if v, ok := raw["page"].(float64); ok {
			meta.Page = int(v)
		}
		if v, ok := raw["limit"].(float64); ok {
			meta.Limit = int(v)
		}
	}

	return output.SuccessEnvelope(raw, meta), nil
}

func (c *Client) Get(path string, query url.Values) (*output.Envelope, error) {
	return c.Do("GET", path, nil, query)
}

func (c *Client) Post(path string, body interface{}) (*output.Envelope, error) {
	return c.Do("POST", path, body, nil)
}

func (c *Client) Put(path string, body interface{}) (*output.Envelope, error) {
	return c.Do("PUT", path, body, nil)
}

func (c *Client) Delete(path string, query url.Values) (*output.Envelope, error) {
	return c.Do("DELETE", path, nil, query)
}

func suggestFix(code int) string {
	switch code {
	case 401:
		return "请先运行 gitlink-cli auth login 登录"
	case 403:
		return "权限不足，请确认账户权限或联系项目管理员"
	case 404:
		return "资源不存在，请检查 owner/repo/id 是否正确"
	case 422:
		return "参数校验失败，请检查请求参数"
	default:
		return ""
	}
}
