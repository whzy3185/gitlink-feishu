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
	}
	for _, replacement := range replacements {
		path = regexp.MustCompile(replacement.pattern).ReplaceAllString(path, replacement.repl)
	}
	return path
}
