package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var openAPIBaseURL = "https://open.feishu.cn/open-apis"

type OpenAPIClient struct {
	BaseURL string
	HTTP    *http.Client
}

type TenantToken struct {
	Value  string
	Expire int
}

type WikiNode struct {
	SpaceID   string `json:"space_id"`
	NodeToken string `json:"node_token"`
	ObjToken  string `json:"obj_token"`
	ObjType   string `json:"obj_type"`
	NodeType  string `json:"node_type"`
	Title     string `json:"title"`
	URL       string `json:"url"`
}

type CreatedDocument struct {
	DocumentID string `json:"document_id"`
	RevisionID int    `json:"revision_id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}

type CreatedBlocks struct {
	RevisionID int `json:"revision_id"`
}

type BitableSearchResult struct {
	RecordID string `json:"record_id,omitempty"`
	Found    bool   `json:"found"`
}

type BitableWriteResult struct {
	RecordID string `json:"record_id,omitempty"`
	Created  bool   `json:"created,omitempty"`
	Updated  bool   `json:"updated,omitempty"`
}

type CreatedTask struct {
	TaskID string `json:"task_id,omitempty"`
}

func NewOpenAPIClient(httpClient *http.Client) OpenAPIClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return OpenAPIClient{
		BaseURL: openAPIBaseURL,
		HTTP:    httpClient,
	}
}

func (c OpenAPIClient) TenantAccessToken(ctx context.Context, appID, appSecret string) (TenantToken, error) {
	body := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return TenantToken{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/auth/v3/tenant_access_token/internal"), bytes.NewReader(reqBody))
	if err != nil {
		return TenantToken{}, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return TenantToken{}, err
	}
	if resp.Code != 0 {
		return TenantToken{}, fmt.Errorf("Feishu tenant token returned code %d: %s", resp.Code, resp.Msg)
	}
	if strings.TrimSpace(resp.TenantAccessToken) == "" {
		return TenantToken{}, fmt.Errorf("Feishu tenant token response missing tenant_access_token")
	}
	return TenantToken{Value: resp.TenantAccessToken, Expire: resp.Expire}, nil
}

func (c OpenAPIClient) GetWikiNode(ctx context.Context, tenantToken string, wikiNodeToken string) (WikiNode, error) {
	query := url.Values{}
	query.Set("token", wikiNodeToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/wiki/v2/spaces/get_node")+"?"+query.Encode(), nil)
	if err != nil {
		return WikiNode{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Node WikiNode `json:"node"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return WikiNode{}, err
	}
	if resp.Code != 0 {
		return WikiNode{}, fmt.Errorf("Feishu wiki get_node returned code %d: %s", resp.Code, resp.Msg)
	}
	if strings.TrimSpace(resp.Data.Node.ObjToken) == "" {
		return WikiNode{}, fmt.Errorf("Feishu wiki node response missing obj_token")
	}
	return resp.Data.Node, nil
}

func (c OpenAPIClient) CreateDocument(ctx context.Context, tenantToken string, folderToken string, title string) (CreatedDocument, error) {
	body := map[string]string{"title": title}
	if strings.TrimSpace(folderToken) != "" {
		body["folder_token"] = strings.TrimSpace(folderToken)
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return CreatedDocument{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/docx/v1/documents"), bytes.NewReader(reqBody))
	if err != nil {
		return CreatedDocument{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Document CreatedDocument `json:"document"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return CreatedDocument{}, err
	}
	if resp.Code != 0 {
		return CreatedDocument{}, fmt.Errorf("Feishu docx create returned code %d: %s", resp.Code, resp.Msg)
	}
	if strings.TrimSpace(resp.Data.Document.DocumentID) == "" {
		return CreatedDocument{}, fmt.Errorf("Feishu docx create response missing document_id")
	}
	return resp.Data.Document, nil
}

func (c OpenAPIClient) CreateBlocks(ctx context.Context, tenantToken string, documentID string, parentBlockID string, blocks []DocBlock) (CreatedBlocks, error) {
	if strings.TrimSpace(parentBlockID) == "" {
		parentBlockID = documentID
	}
	body := map[string]interface{}{"children": blocks}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return CreatedBlocks{}, err
	}
	path := fmt.Sprintf("/docx/v1/documents/%s/blocks/%s/children", url.PathEscape(documentID), url.PathEscape(parentBlockID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), bytes.NewReader(reqBody))
	if err != nil {
		return CreatedBlocks{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			RevisionID int `json:"revision_id"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return CreatedBlocks{}, err
	}
	if resp.Code != 0 {
		return CreatedBlocks{}, fmt.Errorf("Feishu docx create blocks returned code %d: %s", resp.Code, resp.Msg)
	}
	return CreatedBlocks{RevisionID: resp.Data.RevisionID}, nil
}

func (c OpenAPIClient) SearchBitableRecord(ctx context.Context, tenantToken string, appToken string, tableID string, uniqueKey string) (BitableSearchResult, error) {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"conjunction": "and",
			"conditions": []map[string]interface{}{
				{
					"field_name": "unique_key",
					"operator":   "is",
					"value":      []string{uniqueKey},
				},
			},
		},
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return BitableSearchResult{}, err
	}
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/search?page_size=1", url.PathEscape(appToken), url.PathEscape(tableID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), bytes.NewReader(reqBody))
	if err != nil {
		return BitableSearchResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				RecordID string `json:"record_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return BitableSearchResult{}, err
	}
	if resp.Code != 0 {
		return BitableSearchResult{}, fmt.Errorf("Feishu bitable search returned code %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.Data.Items) == 0 || strings.TrimSpace(resp.Data.Items[0].RecordID) == "" {
		return BitableSearchResult{Found: false}, nil
	}
	return BitableSearchResult{RecordID: resp.Data.Items[0].RecordID, Found: true}, nil
}

func (c OpenAPIClient) CreateBitableRecord(ctx context.Context, tenantToken string, appToken string, tableID string, fields map[string]interface{}) (BitableWriteResult, error) {
	body := map[string]interface{}{"fields": fields}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return BitableWriteResult{}, err
	}
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records", url.PathEscape(appToken), url.PathEscape(tableID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), bytes.NewReader(reqBody))
	if err != nil {
		return BitableWriteResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Record struct {
				RecordID string `json:"record_id"`
			} `json:"record"`
			RecordID string `json:"record_id"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return BitableWriteResult{}, err
	}
	if resp.Code != 0 {
		return BitableWriteResult{}, fmt.Errorf("Feishu bitable create returned code %d: %s", resp.Code, resp.Msg)
	}
	recordID := firstNonEmpty(resp.Data.Record.RecordID, resp.Data.RecordID)
	return BitableWriteResult{RecordID: recordID, Created: true}, nil
}

func (c OpenAPIClient) UpdateBitableRecord(ctx context.Context, tenantToken string, appToken string, tableID string, recordID string, fields map[string]interface{}) (BitableWriteResult, error) {
	body := map[string]interface{}{"fields": fields}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return BitableWriteResult{}, err
	}
	path := fmt.Sprintf("/bitable/v1/apps/%s/tables/%s/records/%s", url.PathEscape(appToken), url.PathEscape(tableID), url.PathEscape(recordID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.endpoint(path), bytes.NewReader(reqBody))
	if err != nil {
		return BitableWriteResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Record struct {
				RecordID string `json:"record_id"`
			} `json:"record"`
			RecordID string `json:"record_id"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return BitableWriteResult{}, err
	}
	if resp.Code != 0 {
		return BitableWriteResult{}, fmt.Errorf("Feishu bitable update returned code %d: %s", resp.Code, resp.Msg)
	}
	return BitableWriteResult{RecordID: firstNonEmpty(resp.Data.Record.RecordID, resp.Data.RecordID, recordID), Updated: true}, nil
}

func (c OpenAPIClient) CreateTask(ctx context.Context, tenantToken string, task TaskCandidate) (CreatedTask, error) {
	body := map[string]interface{}{
		"summary":     task.Title,
		"description": task.Description + taskLinkSuffix(task),
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return CreatedTask{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/task/v2/tasks"), bytes.NewReader(reqBody))
	if err != nil {
		return CreatedTask{}, err
	}
	req.Header.Set("Authorization", "Bearer "+tenantToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Task struct {
				GUID   string `json:"guid"`
				TaskID string `json:"task_id"`
			} `json:"task"`
			TaskID string `json:"task_id"`
			GUID   string `json:"guid"`
		} `json:"data"`
	}
	if err := c.doJSON(req, &resp); err != nil {
		return CreatedTask{}, err
	}
	if resp.Code != 0 {
		return CreatedTask{}, fmt.Errorf("Feishu task create returned code %d: %s", resp.Code, resp.Msg)
	}
	return CreatedTask{TaskID: firstNonEmpty(resp.Data.Task.GUID, resp.Data.Task.TaskID, resp.Data.GUID, resp.Data.TaskID)}, nil
}

func (c OpenAPIClient) endpoint(path string) string {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(openAPIBaseURL, "/")
	}
	return base + path
}

func (c OpenAPIClient) doJSON(req *http.Request, target interface{}) error {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var decoded struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		detail := strings.TrimSpace(string(body))
		if err := json.Unmarshal(body, &decoded); err == nil && decoded.Msg != "" {
			detail = fmt.Sprintf("code %d: %s", decoded.Code, decoded.Msg)
		}
		if len(detail) > 300 {
			detail = detail[:300] + "..."
		}
		return fmt.Errorf("Feishu OpenAPI %s %s returned HTTP %d: %s", req.Method, redactOpenAPIPath(req.URL.Path), resp.StatusCode, detail)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse Feishu OpenAPI response: %w", err)
	}
	return nil
}

func redactOpenAPIPath(path string) string {
	replacements := []struct {
		pattern string
		repl    string
	}{
		{`/documents/[^/]+`, `/documents/...`},
		{`/blocks/[^/]+`, `/blocks/...`},
		{`/apps/[^/]+`, `/apps/...`},
		{`/tables/[^/]+`, `/tables/...`},
		{`/records/[^/]+`, `/records/...`},
		{`/tasks/[^/]+`, `/tasks/...`},
	}
	for _, replacement := range replacements {
		path = regexp.MustCompile(replacement.pattern).ReplaceAllString(path, replacement.repl)
	}
	return path
}

func taskLinkSuffix(task TaskCandidate) string {
	links := []string{}
	if task.GitLinkURL != "" {
		links = append(links, "GitLink: "+task.GitLinkURL)
	}
	if task.DocURL != "" {
		links = append(links, "Feishu report: "+task.DocURL)
	}
	if len(links) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(links, "\n")
}

func diagnoseOpenAPIError(err error, category string, targetType string) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	likely := "check Feishu app scopes, resource permissions, IDs, and tenant availability"
	switch category {
	case "task create":
		likely = "grant Task API scopes and verify task creation is enabled for the app"
	case "bitable":
		likely = "grant Base/Bitable scopes and verify app token, table ID, and unique_key field"
	case "docx":
		likely = "grant DocX/Drive scopes and write access to the target document, Wiki node, or folder"
	}
	return fmt.Sprintf("%s failed for %s: %s; likely reason: %s", category, targetType, message, likely)
}
