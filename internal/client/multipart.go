package client

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
)

type MultipartFile struct {
	FieldName string
	FileName  string
	Reader    io.Reader
}

// PostMultipart streams a multipart request without retaining the complete
// upload in memory.
func (c *Client) PostMultipart(path string, fields map[string]string, files []MultipartFile) (*output.Envelope, error) {
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	go func() {
		var writeErr error
		for key, value := range fields {
			if writeErr = form.WriteField(key, value); writeErr != nil {
				break
			}
		}
		if writeErr == nil {
			for _, file := range files {
				field := file.FieldName
				if field == "" {
					field = "file"
				}
				var part io.Writer
				part, writeErr = form.CreateFormFile(field, file.FileName)
				if writeErr != nil {
					break
				}
				_, writeErr = io.Copy(part, file.Reader)
				if writeErr != nil {
					break
				}
			}
		}
		if writeErr == nil {
			writeErr = form.Close()
		}
		_ = writer.CloseWithError(writeErr)
	}()

	normalized := normalizeAPIPath(c.BaseURL, path)
	if shouldAppendJSONSuffix(normalized) {
		normalized += ".json"
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+normalized, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("multipart request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read multipart response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Code: resp.StatusCode, Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return output.SuccessEnvelope(string(body), nil), nil
	}
	return output.SuccessEnvelope(raw, nil), nil
}
