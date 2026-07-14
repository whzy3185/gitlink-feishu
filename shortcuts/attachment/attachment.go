package attachment

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "upload",
			Description: "Upload an attachment",
			Flags: []common.Flag{
				{Name: "file", Short: "f", Usage: "File path to upload", Required: true},
				{Name: "description", Short: "d", Usage: "Attachment description"},
				{Name: "container-id", Usage: "Container model ID"},
				{Name: "container-type", Usage: "Container model type"},
			},
			Run: runUploadAttachment,
		},
		{
			Name:        "delete",
			Description: "Delete an attachment",
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: "Attachment UUID", Required: true},
			},
			Run: runDeleteAttachment,
		},
	}
}

func runUploadAttachment(ctx *common.RuntimeContext) error {
	filePath, err := ctx.RequireArg("file")
	if err != nil {
		return err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("file path points to a directory: %s", filePath)
	}

	fields := map[string]string{}
	if v := ctx.Arg("description"); v != "" {
		fields["description"] = v
	}
	if v := ctx.Arg("container-id"); v != "" {
		fields["container_id"] = v
	}
	if v := ctx.Arg("container-type"); v != "" {
		fields["container_type"] = v
	}

	env, err := ctx.PostMultipart("/attachments", "file", filePath, fields)
	if err != nil {
		return err
	}

	if data, ok := env.Data.(map[string]interface{}); ok {
		data["filename"] = filepath.Base(filePath)
	}

	return ctx.Output(env)
}

func runDeleteAttachment(ctx *common.RuntimeContext) error {
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}

	env, err := ctx.CallAPI("DELETE", fmt.Sprintf("/attachments/%s", id), nil)
	if err != nil {
		return err
	}
	if env == nil {
		return output.Print(output.SuccessEnvelope(map[string]interface{}{
			"id":      id,
			"deleted": true,
		}, nil), ctx.Format)
	}
	return ctx.Output(env)
}
