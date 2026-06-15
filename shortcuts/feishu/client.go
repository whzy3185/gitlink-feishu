package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type WebhookClient struct {
	URL    string
	Secret string
	HTTP   *http.Client
	Now    func() time.Time
}

type WebhookResponse struct {
	StatusCode int    `json:"status_code"`
	Code       int    `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Body       string `json:"body,omitempty"`
}

func (c WebhookClient) Send(ctx context.Context, payload WebhookPayload) (*WebhookResponse, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	if c.Secret != "" {
		ts := timestampSeconds(now())
		payload.Timestamp = strconv.FormatInt(ts, 10)
		payload.Sign = SignCustomBotRequest(ts, c.Secret)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	result := &WebhookResponse{StatusCode: resp.StatusCode, Body: string(respBody)}
	var decoded struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &decoded); err == nil {
		result.Code = decoded.Code
		result.Message = decoded.Msg
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("Feishu webhook returned HTTP %d", resp.StatusCode)
	}
	if decoded.Code != 0 {
		return result, fmt.Errorf("Feishu webhook returned code %d: %s", decoded.Code, decoded.Msg)
	}
	return result, nil
}
