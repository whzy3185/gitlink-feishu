package workflow

func normalizeLang(lang string) string {
	switch lang {
	case "", langEN:
		return langEN
	case langZH:
		return langZH
	default:
		return langEN
	}
}

func message(lang string, key string) string {
	lang = normalizeLang(lang)
	if value, ok := messages[lang][key]; ok {
		return value
	}
	if value, ok := messages[langEN][key]; ok {
		return value
	}
	return key
}

var messages = map[string]map[string]string{
	langEN: {
		"missing_reproduction_steps":     "reproduction_steps",
		"missing_expected_behavior":      "expected_behavior",
		"missing_actual_behavior":        "actual_behavior",
		"missing_version":                "version",
		"missing_os_or_platform":         "os_or_platform",
		"missing_command_output_or_logs": "command_output_or_logs",
		"comment_more_info":              "Thanks for the report. Please add the missing information so maintainers can reproduce and investigate it: %s.",
		"comment_security":               "Thanks for the security report. Please avoid sharing secrets publicly and rotate any exposed credentials. Maintainers should verify the sensitive details in a private channel.",
		"comment_docs":                   "Thanks for the documentation report. Please point to the affected document or example if possible.",
		"comment_default":                "Thanks for the report. Maintainers can use the triage result above to decide the next step.",
		"health_issue_backlog_good":      "Issue backlog is under control.",
		"health_issue_backlog_attention": "Reduce stale or excessive open issues.",
		"health_pr_backlog_good":         "Pull request backlog is under control.",
		"health_pr_backlog_attention":    "Review stale or excessive open pull requests.",
		"health_recent_unknown":          "Recent activity is unknown and was scored conservatively.",
		"health_release_unknown":         "Release status is unknown and was scored conservatively.",
		"health_ci_unknown":              "CI status is unknown and is reported without changing the score.",
		"health_agent_unknown":           "Agent readiness is unknown and was scored conservatively.",
		"rec_maintain":                   "Maintain the current workflow and keep metadata up to date.",
		"rec_reduce_issues":              "Reduce stale issues and add response labels or next actions.",
		"rec_reduce_prs":                 "Review stale pull requests and clarify merge blockers.",
		"rec_restore_activity":           "Create recent maintenance activity or document project status.",
		"rec_release":                    "Publish or document a recent release cadence.",
		"rec_docs":                       "Add or improve README and contribution guidance.",
		"rec_license":                    "Add LICENSE and CONTRIBUTING files for contributor clarity.",
		"rec_agent":                      "Improve agent readiness with stable docs, examples, and machine-readable outputs.",
		"pr_summary_title":               "PR Review Summary",
		"pr_summary_overview":            "Overview",
		"pr_summary_review_focus":        "Review Focus",
		"pr_summary_test_suggestions":    "Test Suggestions",
		"pr_summary_merge_checklist":     "Merge Checklist",
		"pr_summary_reasoning":           "Reasoning",
		"pr_summary_no_focus":            "No specific review focus identified.",
		"pr_summary_no_suggestions":      "No extra test suggestions.",
		"pr_summary_no_checklist":        "No extra merge checklist items.",
		"pr_summary_no_reasoning":        "No additional reasoning.",
	},
	langZH: {
		"missing_reproduction_steps":     "复现步骤",
		"missing_expected_behavior":      "期望行为",
		"missing_actual_behavior":        "实际行为",
		"missing_version":                "版本信息",
		"missing_os_or_platform":         "操作系统或平台",
		"missing_command_output_or_logs": "命令输出或日志",
		"comment_more_info":              "感谢反馈。请补充以下信息，方便维护者复现和定位问题：%s。",
		"comment_security":               "感谢安全反馈。请不要公开扩散密钥或敏感信息，并尽快轮换可能泄露的凭据。维护者应优先通过私密渠道确认细节。",
		"comment_docs":                   "感谢文档反馈。请尽量说明受影响的文档、示例或章节位置。",
		"comment_default":                "感谢反馈。维护者可以根据上面的分诊结果安排下一步处理。",
		"health_issue_backlog_good":      "Issue 积压处于可控状态。",
		"health_issue_backlog_attention": "建议减少长期未处理或数量过多的开放 Issue。",
		"health_pr_backlog_good":         "PR 积压处于可控状态。",
		"health_pr_backlog_attention":    "建议审查长期未处理或数量过多的开放 PR。",
		"health_recent_unknown":          "最近活跃度未知，已按保守方式评分。",
		"health_release_unknown":         "Release 状态未知，已按保守方式评分。",
		"health_ci_unknown":              "CI 状态未知，仅记录为说明，不影响总分。",
		"health_agent_unknown":           "Agent 友好度未知，已按保守方式评分。",
		"rec_maintain":                   "保持当前维护节奏，并持续更新仓库元信息。",
		"rec_reduce_issues":              "减少长期未处理的 Issue，并补充响应标签或下一步动作。",
		"rec_reduce_prs":                 "审查长期未处理的 PR，并明确合并阻塞点。",
		"rec_restore_activity":           "恢复近期维护活动，或在文档中说明项目状态。",
		"rec_release":                    "发布近期版本，或在文档中说明发布节奏。",
		"rec_docs":                       "补充或改进 README 与贡献指南。",
		"rec_license":                    "补充 LICENSE 和 CONTRIBUTING，降低贡献者理解成本。",
		"rec_agent":                      "通过稳定文档、示例和机器可读输出提升 Agent 友好度。",
		"pr_summary_title":               "PR 审阅摘要",
		"pr_summary_overview":            "概览",
		"pr_summary_review_focus":        "审查重点",
		"pr_summary_test_suggestions":    "测试建议",
		"pr_summary_merge_checklist":     "合并检查清单",
		"pr_summary_reasoning":           "判断依据",
		"pr_summary_no_focus":            "未识别到明确审查重点。",
		"pr_summary_no_suggestions":      "暂无额外测试建议。",
		"pr_summary_no_checklist":        "暂无额外合并检查项。",
		"pr_summary_no_reasoning":        "暂无额外判断依据。",
	},
}
