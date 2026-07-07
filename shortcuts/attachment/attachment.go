// Package attachment implements shortcuts for uploading and downloading
// platform attachments (release assets, issue attachments, etc.).
//
// Upload wraps the multipart POST /api/attachments endpoint and returns the
// attachment id that can be fed to `release +create --attachment-ids`;
// download wraps GET /api/attachments/:uuid and streams the file to
// disk, giving the CLI a scriptable path for large-file transfer instead of
// the web UI.
package attachment

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// Shortcuts returns attachment upload/download shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}

	return []*common.Shortcut{
		{
			Name:        "upload",
			Description: tr.T("cmd.attachment.upload.short"),
			Long:        tr.T("cmd.attachment.upload.long"),
			Flags: []common.Flag{
				{Name: "file", Short: "f", Usage: tr.T("flag.attachment.file"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.attachment.description")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				file, err := ctx.RequireArg("file")
				if err != nil {
					return err
				}
				info, err := os.Stat(file)
				if err != nil {
					return fmt.Errorf("cannot access file %q: %w", file, err)
				}
				if info.IsDir() {
					return fmt.Errorf("%q is a directory, expected a file", file)
				}
				fields := map[string]string{
					"description": ctx.Arg("description"),
				}
				env, err := ctx.Client.PostMultipartFile("/attachments", file, "file", fields)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "download",
			Description: tr.T("cmd.attachment.download.short"),
			Long:        tr.T("cmd.attachment.download.long"),
			Flags: []common.Flag{
				{Name: "id", Short: "i", Usage: tr.T("flag.attachment.id"), Required: true},
				{Name: "output", Short: "o", Usage: tr.T("flag.attachment.output")},
			},
			Run: func(ctx *common.RuntimeContext) error {
				id, err := ctx.RequireArg("id")
				if err != nil {
					return err
				}
				dest := ctx.Arg("output")
				if dest == "" {
					dest = id
				}
				n, err := ctx.Client.DownloadFile("/attachments/"+id, dest)
				if err != nil {
					return err
				}
				return ctx.OutputData(map[string]interface{}{
					"file":  filepath.Clean(dest),
					"bytes": n,
				})
			},
		},
	}
}
