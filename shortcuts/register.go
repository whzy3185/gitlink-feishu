package shortcuts

import (
	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/branch"
	"github.com/gitlink-org/gitlink-cli/shortcuts/ci"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/compare"
	"github.com/gitlink-org/gitlink-cli/shortcuts/dataset"
	"github.com/gitlink-org/gitlink-cli/shortcuts/explore"
	"github.com/gitlink-org/gitlink-cli/shortcuts/file"
	"github.com/gitlink-org/gitlink-cli/shortcuts/health"
	"github.com/gitlink-org/gitlink-cli/shortcuts/ignore"
	"github.com/gitlink-org/gitlink-cli/shortcuts/issue"
	"github.com/gitlink-org/gitlink-cli/shortcuts/label"
	"github.com/gitlink-org/gitlink-cli/shortcuts/license"
	"github.com/gitlink-org/gitlink-cli/shortcuts/member"
	"github.com/gitlink-org/gitlink-cli/shortcuts/milestone"
	"github.com/gitlink-org/gitlink-cli/shortcuts/org"
	"github.com/gitlink-org/gitlink-cli/shortcuts/pipeline"
	"github.com/gitlink-org/gitlink-cli/shortcuts/pm"
	"github.com/gitlink-org/gitlink-cli/shortcuts/pr"
	"github.com/gitlink-org/gitlink-cli/shortcuts/profile"
	"github.com/gitlink-org/gitlink-cli/shortcuts/release"
	"github.com/gitlink-org/gitlink-cli/shortcuts/repo"
	"github.com/gitlink-org/gitlink-cli/shortcuts/search"
	"github.com/gitlink-org/gitlink-cli/shortcuts/snippet"
	"github.com/gitlink-org/gitlink-cli/shortcuts/user"
	"github.com/gitlink-org/gitlink-cli/shortcuts/webhook"
	"github.com/gitlink-org/gitlink-cli/shortcuts/wiki"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

// RegisterAll mounts all shortcut groups onto the root command.
func RegisterAll(root *cobra.Command, translators ...*i18n.Translator) {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	groups := map[string][]*common.Shortcut{
		"repo":      repo.Shortcuts(tr),
		"issue":     issue.Shortcuts(tr),
		"pr":        pr.Shortcuts(tr),
		"release":   release.Shortcuts(tr),
		"branch":    branch.Shortcuts(tr),
		"org":       org.Shortcuts(tr),
		"user":      user.Shortcuts(tr),
		"search":    search.Shortcuts(tr),
		"ci":        ci.Shortcuts(tr),
		"milestone": milestone.Shortcuts(),
		"label":     label.Shortcuts(),
		"file":      file.Shortcuts(),
		"webhook":   webhook.Shortcuts(tr),
		"member":    member.Shortcuts(),
		"snippet":   snippet.Shortcuts(),
		"wiki":      wiki.Shortcuts(),
		"compare":   compare.Shortcuts(),
		"dataset":   dataset.Shortcuts(tr),
		"explore":   explore.Shortcuts(tr),
		"health":    health.Shortcuts(tr),
		"ignore":    ignore.Shortcuts(),
		"license":   license.Shortcuts(),
		"pipeline":  pipeline.Shortcuts(),
		"pm":        pm.Shortcuts(),
		"profile":   profile.Shortcuts(tr),
		"workflow":  workflow.Shortcuts(),
	}

	descriptions := map[string]string{
		"repo":      tr.T("cmd.repo.short"),
		"issue":     tr.T("cmd.issue.short"),
		"pr":        tr.T("cmd.pr.short"),
		"release":   tr.T("cmd.release.short"),
		"branch":    tr.T("cmd.branch.short"),
		"org":       tr.T("cmd.org.short"),
		"user":      tr.T("cmd.user.short"),
		"search":    tr.T("cmd.search.short"),
		"ci":        tr.T("cmd.ci.short"),
		"milestone": "Milestone operations",
		"label":     "Issue label (tag) operations",
		"file":      "File operations",
		"webhook":   tr.T("cmd.webhook.short"),
		"member":    "Project member operations",
		"snippet":   "Local code snippet management",
		"wiki":      "Wiki operations",
		"compare":   "Compare branches, tags, or commits",
		"dataset":   tr.T("cmd.dataset.short"),
		"explore":   "Explore pinned projects and categories",
		"health":    "Project health data collection",
		"ignore":    "Gitignore template operations",
		"license":   "License operations",
		"pipeline":  "Pipeline operations",
		"pm":        "Project management operations",
		"profile":   tr.T("cmd.profile.short"),
		"workflow":  "AI agent workflow analysis",
	}

	for name, shortcuts := range groups {
		groupCmd := &cobra.Command{
			Use:   name,
			Short: descriptions[name],
		}
		common.MountShortcuts(groupCmd, shortcuts, tr)
		root.AddCommand(groupCmd)
	}
}
