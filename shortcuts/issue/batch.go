package issue

import (
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const closeIssueStatusID = 5

func newBatchCloseShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-close",
		Description: tr.T("cmd.issue.batch_close.short"),
		Flags:       batchStateFlags(tr),
		Run:         func(ctx *common.RuntimeContext) error { return runBatchStateChange(ctx, "close", closeIssueStatusID) },
	}
}
