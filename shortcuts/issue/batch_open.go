package issue

import (
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const openIssueStatusID = 1

func newBatchOpenShortcut(tr *i18n.Translator) *common.Shortcut {
	return &common.Shortcut{
		Name:        "batch-open",
		Description: tr.T("cmd.issue.batch_open.short"),
		Flags:       batchStateFlags(tr),
		Run:         func(ctx *common.RuntimeContext) error { return runBatchStateChange(ctx, "open", openIssueStatusID) },
	}
}
