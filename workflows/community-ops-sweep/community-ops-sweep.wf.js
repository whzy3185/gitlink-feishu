export const meta = {
  name: 'community-ops-sweep',
  description: 'Incremental community ops sweep — triage new issues (R1), suggest owners (R2), link merged PRs→issues (R5), generate weekly report + release notes (R3/R4), upload report to Wiki. Default dry-run; --apply to write.',
  phases: [
    { title: 'Collect', detail: 'Python engine gathers new issues + merged/open PRs + contributor stats since checkpoint' },
    { title: 'Triage', detail: 'LLM agents classify, suggest owners, link PRs — in parallel' },
    { title: 'Summarize', detail: 'LLM generates executive summary from collected data' },
    { title: 'Plan', detail: 'Python engine builds deterministic write plan + generates seven-section report' },
    { title: 'Apply', detail: 'Execute (or dry-run) write plan + publish to Wiki + advance checkpoint' },
  ],
}

const cfg = args || {}
const OWNER = cfg.owner
const REPO = cfg.repo
const APPLY = cfg.apply === true
const REPORT_ISSUE = cfg.reportIssue || ''
const RELEASE_ID = cfg.releaseId || ''
const SINCE = cfg.since || ''
const PUBLISH_WIKI = cfg.publishWiki === true
const WIKI_DIR = cfg.wikiDir || '周报专区'
const REPORT_FLAG = REPORT_ISSUE ? `--report-issue ${REPORT_ISSUE}` : ''
const RELEASE_FLAG = RELEASE_ID ? `--release-id ${RELEASE_ID}` : ''
const WIKI_FLAG = PUBLISH_WIKI ? `--publish-wiki --wiki-dir ${WIKI_DIR}` : ''

const ENGINE = 'examples/workflows/community-ops-sweep/scripts/community_ops_sweep.py'
const CANDIDATES = '/tmp/sweep-candidates.json'
const TRIAGE = '/tmp/sweep-triage.json'
const OWNERS = '/tmp/sweep-owners.json'
const LINKS = '/tmp/sweep-links.json'
const PLAN = '/tmp/sweep-plan.json'
const SUMMARY = '/tmp/sweep-summary.json'
const SUMMARY_ARG = `--summary ${SUMMARY}`

const SINCE_ARG = SINCE ? `--since ${SINCE}` : ''

if (!OWNER || !REPO) {
  throw new Error('Missing required args: owner, repo')
}

phase('Collect')
log(`Collecting since=${SINCE || 'checkpoint'} for ${OWNER}/${REPO}`)
const collectOut = await agent(
  `Run the sweep engine collect:\n` +
  `  python3 ${ENGINE} collect --owner ${OWNER} --repo ${REPO} ${SINCE_ARG} --out ${CANDIDATES}\n` +
  `Then Read ${CANDIDATES} and return a compact summary: how many new_issues, merged_prs, open_issues, and the tag_map keys. Also list each issue's number, title and current labels.`,
  { label: 'collect', phase: 'Collect' }
)
log(`Collect result: ${collectOut}`)

// -----------------------------------------------------------------------
// Phase Triage — three LLM agents, independent → parallel
// -----------------------------------------------------------------------
phase('Triage')
log('R1 classify + R2 suggest owners + R5 link PRs — in parallel')
const triagePrompts = [
  // R1: classify new issues
  () => agent(
     `## Task: Classify new issues for triage (R1)\n\n` +
     `1. Read the file ${CANDIDATES}. It contains:\n` +
     `   - "issues": new issues with fields number, title, description, tags, labels\n` +
     `   - "tag_map": available label names → IDs. ONLY use labels from tag_map keys.\n\n` +
     `2. For EACH issue, determine:\n` +
     `   - tag_names: labels to ADD (pick from tag_map keys, e.g. "bug","feature","enhancement","question","docs","P1","P2","P3")\n` +
     `   - remove_tag_names: labels to REMOVE if any current ones are wrong\n` +
     `   - recommended_action: one of "schedule_fix","duplicate","answered","spam","invalid". Use "schedule_fix" for normal issues.\n` +
     `     (duplicate/answered/spam/invalid → will be auto-closed)\n\n` +
     `3. Write your decision as a single-line JSON array to ${TRIAGE} using Bash:\n` +
     `   cat > ${TRIAGE} << 'EOD'\n   [{"issue":"<num>","tag_names":["<name>"],"remove_tag_names":["<name>"],"recommended_action":"<action>"}]\n   EOD\n` +
     `   Make sure each issue has an entry if there are decisions. Skip issues that need no changes.\n\n` +
     `4. Confirm: cat ${TRIAGE}`,
     { label: 'triage:R1-classify', phase: 'Triage' }
  ),
  // R2: suggest owners
  () => agent(
     `## Task: Suggest owners for new issues (R2)\n\n` +
     `1. Read the file ${CANDIDATES}.\n\n` +
     `2. For EACH issue, suggest 1-3 likely owners based on:\n` +
     `   - The issue content (title, description)\n` +
     `   - Author logins of recent merged PRs (in "merged_prs")\n` +
     `   - General domain familiarity\n\n` +
     `3. Output: write a JSON array to ${OWNERS} using Bash:\n` +
     `   cat > ${OWNERS} << 'EOD'\n   [{"issue":"<num>","suggested_owners":["<login>"],"evidence":"<why>"}]\n   EOD\n` +
     `   If you cannot suggest anyone for an issue, omit it.\n\n` +
     `4. Confirm: cat ${OWNERS}`,
     { label: 'triage:R2-owners', phase: 'Triage' }
  ),
  // R5: link merged PRs → issues
  () => agent(
     `## Task: Link merged PRs to issues (R5)\n\n` +
     `1. Read the file ${CANDIDATES}. Focus on the "merged_prs" list.\n\n` +
     `2. For EACH merged PR, parse its "description" (body) for patterns like:\n` +
     `   - "fixes #N", "closes #N", "resolves #N"\n` +
     `   - Chinese equivalents: "修复 #N", "关闭 #N", "解决 #N"\n` +
     `   - Semantic references that clearly identify a specific issue number\n\n` +
     `3. Output: write a JSON array to ${LINKS} using Bash:\n` +
     `   cat > ${LINKS} << 'EOD'\n   [{"pr":"<num>","linked_issue_numbers":["<num>"]}]\n   EOD\n` +
     `   If no PR references any issue, write an empty array [].\n\n` +
     `4. Confirm: cat ${LINKS}`,
     { label: 'triage:R5-link', phase: 'Triage' }
  ),
]

await Promise.all(triagePrompts.map(fn => fn()))
log('Triage complete — decision files written')

// -----------------------------------------------------------------------
// Phase Summarize — LLM generates executive summary
// -----------------------------------------------------------------------
phase('Summarize')
log('Generating executive summary from collected data')
const summaryOut = await agent(
  `## Task: Generate weekly report executive summary\n\n` +
  `1. Read the file ${CANDIDATES}. It contains:\n` +
  `   - "issues": new issues this period (number, title, labels, author_login, state)\n` +
  `   - "merged_prs": merged PRs (number, title, author_login)\n` +
  `   - "open_prs": pending PRs\n` +
  `   - "current_counts": issue/PR counts\n` +
  `   - "contributors": top contributors with issue/PR counts\n` +
  `   - "trends": trend arrows (↑↓→) compared to last period\n\n` +
  `2. Write a 3-5 sentence executive summary in Chinese to ${SUMMARY} using Bash:\n` +
  `   cat > ${SUMMARY} << 'EOD'\n   {"summary": "本周社区新增 N 个 Issue，合并 M 个 PR。……"}\n   EOD\n\n` +
  `3. The summary should cover:\n` +
  `   - Overall activity level (busy/calm/normal)\n` +
  `   - Key highlights (major features merged, important bugs fixed)\n` +
  `   - Notable trends (↑ or ↓ compared to last week)\n` +
  `   - Any concerns or action items\n\n` +
  `4. Confirm: cat ${SUMMARY}`,
  { label: 'summarize', phase: 'Summarize' }
)
log(`Summary: ${summaryOut}`)

// -----------------------------------------------------------------------
// Phase Plan — deterministic engine assembles write plan + generates report
// -----------------------------------------------------------------------
phase('Plan')
const planOut = await agent(
  `Run the sweep engine plan:\n` +
  `  python3 ${ENGINE} plan --candidates ${CANDIDATES} --triage ${TRIAGE} --owners ${OWNERS} --links ${LINKS} ${SUMMARY_ARG} --out ${PLAN}\n` +
  `Then Read ${PLAN} and summarize: how many writes, by op (update_tags/update_status/comment/update_assigner), which issues affected. Show the writes table.`,
  { label: 'plan', phase: 'Plan' }
)
log(`Plan: ${planOut}`)

// -----------------------------------------------------------------------
// Phase Apply — execute (or dry-run) + checkpoint
// -----------------------------------------------------------------------
phase('Apply')
const mode = APPLY ? '--apply' : ''
const applyOut = await agent(
  APPLY
    ? `Execute the write plan (--apply):\n  python3 ${ENGINE} apply --plan ${PLAN} --owner ${OWNER} --repo ${REPO} --apply ${REPORT_FLAG} ${RELEASE_FLAG} ${WIKI_FLAG}\nThen: python3 ${ENGINE} checkpoint --owner ${OWNER} --repo ${REPO} --plan ${PLAN}\nReturn a summary of writes executed and where the report was published.`
    : `Dry-run the write plan (no writes executed):\n  python3 ${ENGINE} apply --plan ${PLAN} --owner ${OWNER} --repo ${REPO} ${REPORT_FLAG} ${RELEASE_FLAG} ${WIKI_FLAG}\nSummarize what WOULD be written.`,
  { label: 'apply', phase: 'Apply' }
)
log(`Apply: ${applyOut}`)

return {
  mode: APPLY ? 'applied' : 'dry-run',
  owner: OWNER, repo: REPO,
  collect: collectOut,
  plan: planOut,
  apply: applyOut,
}
