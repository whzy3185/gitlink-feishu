package attachment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns attachment upload/delete shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "upload",
			Description: "Upload an attachment file",
			Flags: []common.Flag{
				{Name: "file", Short: "f", Usage: "Local file path to upload", Required: true},
				{Name: "description", Short: "d", Usage: "Attachment description"},
				{Name: "container-id", Usage: "Optional container model ID"},
				{Name: "container-type", Usage: "Optional container model type"},
				{Name: "dry-run", Usage: "Preview the multipart fields without uploading the file", Bool: true, Default: "false"},
			},
			Run: runUpload,
		},
		{
			Name:        "delete",
			Description: "Delete an attachment by UUID",
			Flags: []common.Flag{
				{Name: "uuid", Short: "u", Usage: "Attachment UUID", Required: true},
				{Name: "dry-run", Usage: "Preview the delete request without deleting the attachment", Bool: true, Default: "false"},
			},
			Run: runDelete,
		},
	}
}

func runUpload(ctx *common.RuntimeContext) error {
	filePath, err := ctx.RequireArg("file")
	if err != nil {
		return err
	}
	fields := attachmentFields(ctx)
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":  true,
			"action":   "upload_attachment",
			"method":   "POST",
			"path":     "/attachments",
			"file":     filePath,
			"filename": filepath.Base(filePath),
			"fields":   fields,
		})
	}
	env, err := uploadAttachment(ctx, filePath, fields)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runDelete(ctx *common.RuntimeContext) error {
	uuid, err := ctx.RequireArg("uuid")
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/attachments/%s", uuid)
	if parseBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(map[string]interface{}{
			"dry_run": true,
			"action":  "delete_attachment",
			"method":  "DELETE",
			"path":    path,
		})
	}
	env, err := ctx.CallAPI("DELETE", path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func attachmentFields(ctx *common.RuntimeContext) map[string]string {
	fields := map[string]string{}
	for _, name := range []string{"description", "container-id", "container-type"} {
		if value := ctx.Arg(name); value != "" {
			fields[apiFieldName(name)] = value
		}
	}
	return fields
}

func apiFieldName(flagName string) string {
	switch flagName {
	case "container-id":
		return "container_id"
	case "container-type":
		return "container_type"
	default:
		return flagName
	}
}

func uploadAttachment(ctx *common.RuntimeContext, filePath string, fields map[string]string) (*output.Envelope, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open attachment file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create multipart file field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("read attachment file: %w", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("write multipart field %s: %w", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := apiURL(ctx.Client.BaseURL, "/attachments")
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := ctx.Client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respData)))
	}

	var parsed interface{}
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return output.SuccessEnvelope(string(respData), nil), nil
	}
	if data, ok := parsed.(map[string]interface{}); ok {
		if status, ok := data["status"].(float64); ok && status != 0 && status != 1 && status != 200 {
			message, _ := data["message"].(string)
			return nil, fmt.Errorf("[%v] %s", status, message)
		}
	}
	return output.SuccessEnvelope(parsed, nil), nil
}

func apiURL(baseURL, path string) string {
	fullPath := path
	if !strings.HasSuffix(fullPath, ".json") {
		fullPath += ".json"
	}
	return strings.TrimRight(baseURL, "/") + fullPath
}

func parseBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}
