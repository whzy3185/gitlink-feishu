# gitlink-cli

[![GitLink](https://img.shields.io/badge/GitLink-Gitlink%2Fgitlink--cli-green)](https://www.gitlink.org.cn/Gitlink/gitlink-cli)
[![License](https://img.shields.io/badge/License-MulanPSL--2.0-blue.svg)](https://license.coscl.org.cn/MulanPSL2)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue.svg)](https://golang.org)
[![npm version](https://img.shields.io/npm/v/@gitlink-ai/cli.svg)](https://www.npmjs.com/package/@gitlink-ai/cli)

The official [GitLink](https://www.gitlink.org.cn) CLI tool — built for humans and AI Agents. Supports **macOS, Linux, and Windows**. Covers repository management, issue tracking, pull requests, webhooks, member collaboration, CI/CD, and AI-powered workflows, with 40+ commands and AI Agent [Skills](./skills/).

**[中文文档](./README.zh-CN.md)**

[Install](#installation--quick-start) · [AI Agent Skills](#ai-agent-skills) · [Auth](#configure--use) · [Commands](#usage-examples) · [Contributing](#related-projects)

## Contributors

<div style="display: flex; gap: 16px; flex-wrap: wrap; align-items: flex-start;">
<div align="center">
  <a href="https://www.gitlink.org.cn/wangyue111" title="wangyue111"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/W/43_254_70/120.png" width="40" height="40" alt="wangyue111" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/wangyue111">wangyue111</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/wbtiger" title="tigerwang"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/T/14_168_39/120.png" width="40" height="40" alt="wbtiger" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/wbtiger">wbtiger</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/Mengz" title="Mengz"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/M/166_152_185/120.png" width="40" height="40" alt="Mengz" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/Mengz">Mengz</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/yangsai" title="杨赛"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/Y/94_150_149/120.png" width="40" height="40" alt="yangsai" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/yangsai">yangsai</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/mengcheng" title="camelliamc"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/M/206_114_54/120.png" width="40" height="40" alt="mengcheng" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/mengcheng">mengcheng</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/muel" title="赵奕程"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/Z/144_206_212/120.png" width="40" height="40" alt="muel" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/muel">muel</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/Leo77" title="Leo77"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/L/173_120_149/120.png" width="40" height="40" alt="Leo77" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/Leo77">Leo77</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/yingjie" title="yingjie"><img src="https://www.gitlink.org.cn/images/avatars/User/145288?t=1765791899" width="40" height="40" alt="yingjie" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/yingjie">yingjie</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/topshare" title="Kevin Zhang"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/K/65_152_142/120.png" width="40" height="40" alt="topshare" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/topshare">topshare</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/dtwdtw" title="dtwdtw"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/D/53_166_51/120.png" width="40" height="40" alt="dtwdtw" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/dtwdtw">dtwdtw</a></sub>
</div>
<div align="center">
  <a href="https://www.gitlink.org.cn/recorder" title="recorder"><img src="https://www.gitlink.org.cn/system/lets/letter_avatars/2/R/141_201_87/120.png" width="40" height="40" alt="recorder" style="border-radius: 50%;"></a>
  <br><sub><a href="https://www.gitlink.org.cn/recorder">recorder</a></sub>
</div>
</div>

## Why gitlink-cli?

- **Agent-Native Design** — Structured [Skills](./skills/) out of the box, compatible with Claude Code, OpenClaw, and other AI platforms — Agents can operate GitLink with zero extra setup
- **Wide Coverage** — Repository, Issue, PR, Webhook, Member, Branch, Release, CI, Pipeline, Org, Search, and User workflows are covered by high-level commands
- **AI-Friendly & Optimized** — Every command is tested with real Agents, featuring concise parameters, smart defaults, and structured output
- **Cross-Platform** — Runs on macOS, Linux, and Windows (x64/arm64), install via `npm install -g @gitlink-ai/cli` in one command, binary auto-downloaded
- **Open Source, Zero Barriers** — MulanPSL-2.0 license, ready to use, just `npm install`
- **Up and Running in 3 Minutes** — Interactive login or `GITLINK_TOKEN` env var, from install to first API call in just 3 steps
- **Secure & Controllable** — OS-native keychain credential storage, `GITLINK_TOKEN` env var for CI/CD & non-interactive environments, auto git remote context resolution
- **Three-Layer Architecture** — Shortcuts (human & AI friendly) → Raw API (full coverage) → Config (configuration management)

## Features

| Category | Capabilities |
|----------|-------------|
| 📦 Repo | List, create, fork, delete repositories, view repo info |
| 🐛 Issue | Create, update, close, batch close, comment on issues |
| 🔖 Label | Create, list, update, delete issue labels |
| 🔀 PR | Create, merge, review pull requests, view changed files |
| 👥 Member | List, add, remove repository members, change roles, create and accept invite links |
| 🌿 Branch | Create, delete, list, protect, unprotect branches |
| 🏷️ Release | Create, view, delete releases |
| 🏢 Org | Manage organizations, members, teams |
| 🔧 CI | View builds, logs, CI/CD operations |
| ⚙️ Pipeline | Run, inspect, enable, disable, delete pipeline workflows and logs |
| 🔔 Webhook | Manage repo webhooks and test deliveries |
| 🔍 Search | Search repositories, users |
| 👤 User | View user profiles and info |
| 📋 PM | Sprint management, kanban boards, weekly reports |
| 🤖 Workflow | AI-powered issue triage, PR review, release notes |

## Installation & Quick Start

### Requirements

- Node.js 14+ (`npm`/`npx`) — for npm installation
- Supported platforms: macOS, Linux, Windows (x64/arm64)
- Go 1.26+ — only required for building from source

### Quick Start (Human Users)

> **Note for AI assistants:** If you are an AI Agent helping the user with installation, jump directly to [Quick Start (AI Agent)](#quick-start-ai-agent), which contains all the steps you need to complete.

#### Install

**From npm (recommended):**

```bash
# One command: installs CLI binary + AI Agent Skills
npm install -g @gitlink-ai/cli
```

The binary is auto-downloaded for your platform during `postinstall`. No extra steps needed.

**From source:**

Requires Go 1.26+.

```bash
git clone https://www.gitlink.org.cn/Gitlink/gitlink-cli.git
cd gitlink-cli
make install
```

> **Windows users:** Run `npm install -g @gitlink-ai/cli` in PowerShell or CMD. For building from source, use `go install .` instead of `make install`.

#### Configure & Use

```bash
# 1. Configure (one-time, interactive guided setup)
gitlink-cli config init

# 2. Log in (choose one)
gitlink-cli auth login            # Username/password (recommended)
gitlink-cli auth login --token    # Or paste a private token
export GITLINK_TOKEN="your-token" # Or set env var (for CI/CD, non-interactive environments)

# 3. Start using
gitlink-cli repo +list
```

### Quick Start (AI Agent)

> The following steps are for AI Agents. Some steps require the user to complete actions in a browser.

**Step 1 — Install**

```bash
# One command: CLI binary + all Skills auto-installed
npm install -g @gitlink-ai/cli
```

**Step 2 — Configure**

```bash
gitlink-cli config init
```

**Step 3 — Login**

For interactive environments:
```bash
gitlink-cli auth login
```

For non-interactive environments (CI/CD, Trae sandbox, MCP, etc.):
```bash
export GITLINK_TOKEN="your-private-token"
```

> To get a private token, go to GitLink web → Settings → Private Tokens.

**Step 4 — Verify**

```bash
gitlink-cli user +me
```

## Usage Examples

### Repository Operations

```bash
# List repositories
gitlink-cli repo +list

# View repository info
gitlink-cli repo +info --owner Gitlink --repo forgeplus

# Read repository README
gitlink-cli repo +readme --owner Gitlink --repo forgeplus --ref master

# Create a repository
gitlink-cli repo +create -n my-project -d "Project description"

# Fork a repository
gitlink-cli repo +fork --owner Gitlink --repo forgeplus
```

### Webhook Management

```bash
# List webhooks
gitlink-cli webhook +list --owner Gitlink --repo forgeplus

# Create a webhook
gitlink-cli webhook +create --owner Gitlink --repo forgeplus \
  --url https://example.com/hook --events push,create

# Test a webhook
gitlink-cli webhook +test --owner Gitlink --repo forgeplus --id 68

# View webhook delivery tasks
gitlink-cli webhook +tasks --owner Gitlink --repo forgeplus --id 68
```

### Member Management

```bash
# List repository members
gitlink-cli member +list --owner Gitlink --repo forgeplus

# Add a member
gitlink-cli member +add --owner Gitlink --repo forgeplus --user-id 101

# Preview batch add without changing data
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --user-ids 101,102 --dry-run

# Batch add members from a CSV file
gitlink-cli member +batch-add --owner Gitlink --repo forgeplus --from members.csv

# Change a member role
gitlink-cli member +role --owner Gitlink --repo forgeplus --user-id 101 --role Developer

# Create an invite link
gitlink-cli member +invite-link --owner Gitlink --repo forgeplus --role developer --apply true
```

### Issue Management

```bash
# List issues
gitlink-cli issue +list --owner Gitlink --repo forgeplus

# Create an issue
gitlink-cli issue +create --owner Gitlink --repo forgeplus -t "Bug: Login failed" -b "Steps to reproduce..."

# Create an issue with metadata
gitlink-cli issue +create --owner Gitlink --repo forgeplus -t "Bug: Login failed" --priority-id 3 --tag-ids 4,5 --assigner-ids 7

# View an issue
gitlink-cli issue +view --owner Gitlink --repo forgeplus -i 123

# Update issue metadata
gitlink-cli issue +update --owner Gitlink --repo forgeplus --number 123 --priority-id 4 --branch bugfix/login --due-date 2026-06-15

# Close an issue
gitlink-cli issue +close --owner Gitlink --repo forgeplus -i 123

# Preview batch close without changing data
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --numbers 123,124 --dry-run

# Batch close issues from a CSV file
gitlink-cli issue +batch-close --owner Gitlink --repo forgeplus --from issues.csv

# Add a comment
gitlink-cli issue +comment --owner Gitlink --repo forgeplus -i 123 -b "Fixed"

# List issue assigners
gitlink-cli issue +assigners --owner Gitlink --repo forgeplus

# List issue authors
gitlink-cli issue +authors --owner Gitlink --repo forgeplus

# List issue priorities
gitlink-cli issue +priorities --owner Gitlink --repo forgeplus

# List issue tags
gitlink-cli issue +tags --owner Gitlink --repo forgeplus --only-name

# List issue statuses
gitlink-cli issue +statuses --owner Gitlink --repo forgeplus
```

`issue +view`, `issue +update`, `issue +close`, and `issue +comment` prefer
`--number` / `-n` for the issue number shown in the web URL. `--id` / `-i`
is accepted as a compatibility alias for the same web issue number, not the
global database ID.

### Label Management

```bash
# List issue labels
gitlink-cli label +list --owner Gitlink --repo forgeplus

# Filter labels by keyword
gitlink-cli label +list --owner Gitlink --repo forgeplus -k bug

# Create a label (color defaults to #1E90FF)
gitlink-cli label +create --owner Gitlink --repo forgeplus -n bug -d "Something is broken" -c "#FF0000"

# Update a label (unspecified fields are preserved)
gitlink-cli label +update --owner Gitlink --repo forgeplus -i 42 -c "#00FF00"

# Delete a label
gitlink-cli label +delete --owner Gitlink --repo forgeplus -i 42
```

### Pull Requests

```bash
# List PRs
gitlink-cli pr +list --owner Gitlink --repo forgeplus

# Create a PR (same-repo branch)
gitlink-cli pr +create --owner Gitlink --repo forgeplus -t "feat: Search feature" --head feature/search --base master

# Create a PR (from a fork)
gitlink-cli pr +create --owner Gitlink --repo forgeplus -t "feat: New feature" --head your_username/forgeplus:feature/my-feature --base master

# View a PR
gitlink-cli pr +view --owner Gitlink --repo forgeplus -i 42

# Merge a PR
gitlink-cli pr +merge --owner Gitlink --repo forgeplus -i 42

# Reopen a closed PR
gitlink-cli pr +reopen --owner Gitlink --repo forgeplus -i 42

# View changed files
gitlink-cli pr +files --owner Gitlink --repo forgeplus -i 42

# List PR patchset versions
gitlink-cli pr +versions --owner Gitlink --repo forgeplus -i 42

# View a patchset version diff
gitlink-cli pr +version-diff --owner Gitlink --repo forgeplus -i 42 --version-id 16040

# List PR reviews
gitlink-cli pr +reviews --owner Gitlink --repo forgeplus -i 42

# Create a PR review (with dry-run preview)
gitlink-cli pr +review --owner Gitlink --repo forgeplus -i 42 --status approved -c "LGTM" --dry-run
gitlink-cli pr +review --owner Gitlink --repo forgeplus -i 42 --status approved -c "LGTM"
```

### Branch Management

```bash
# List branches
gitlink-cli branch +list --owner Gitlink --repo forgeplus

# Create a branch
gitlink-cli branch +create --name feature/new-feature

# Delete a branch
gitlink-cli branch +delete --name feature/old-feature

# Protect a branch
gitlink-cli branch +protect --name main

# Remove branch protection
gitlink-cli branch +unprotect --name main
```

### Release Management

```bash
# List releases
gitlink-cli release +list --owner Gitlink --repo forgeplus

# Create a release
gitlink-cli release +create --owner Gitlink --repo forgeplus -t v1.0.0 -n "v1.0.0 Stable" -b "Changelog..."

# View a release
gitlink-cli release +view --owner Gitlink --repo forgeplus -i <version_id>
```

### CI/CD Operations

```bash
# List builds
gitlink-cli ci +list --owner Gitlink --repo forgeplus

# View build log
gitlink-cli ci +log --owner Gitlink --repo forgeplus -i <build_id>

# Restart a build
gitlink-cli ci +restart --owner Gitlink --repo forgeplus -i <build_id>
```

### Pipeline Operations

```bash
# List platform pipelines
gitlink-cli pipeline +list --owner-id 123 --page 1 --limit 20

# List repository pipeline runs
gitlink-cli pipeline +runs --owner Gitlink --repo forgeplus --ref master --workflow build.yml

# Start a pipeline workflow, previewing the request first
gitlink-cli pipeline +run --owner Gitlink --repo forgeplus --ref master --workflow build.yml --dry-run

# Inspect pipeline details and logs
gitlink-cli pipeline +view --owner Gitlink --repo forgeplus --id 7
gitlink-cli pipeline +logs --owner Gitlink --repo forgeplus --run-id 99 --id 7 --index 43
gitlink-cli pipeline +results --owner Gitlink --repo forgeplus --run-id 99

# Toggle or delete pipeline workflows, previewing destructive writes first
gitlink-cli pipeline +disable --owner Gitlink --repo forgeplus --id 7 --workflow build.yml --dry-run
gitlink-cli pipeline +delete --owner Gitlink --repo forgeplus --id 7 --dry-run
```

### Search

```bash
# Search repositories
gitlink-cli search +repos -k "machine learning"

# Search users
gitlink-cli search +users -k "zhangsan"
```

### Workflow Agent Commands

`workflow` provides rule-based repository analysis for maintainers and AI Agents. It currently supports:

- `workflow +triage`
- `workflow +health`
- `workflow +pr-summary`
- `workflow +repo-report`

`workflow +pr-summary` defaults to `table` when `--format` is omitted.
`workflow +repo-report` defaults to `markdown` when `--format` is omitted.

Examples:

```bash
# Triage with local parameters
gitlink-cli workflow +triage --title "Install failed on Windows" --body "go install failed with error" --format table

# Triage with JSON output
gitlink-cli workflow +triage --title "Token leaked in logs" --body "The access token appears in command output" --format json

# Triage with Chinese markdown output
gitlink-cli workflow +triage \
  --title "安装失败，无法登录" \
  --body "运行命令时报错" \
  --lang zh-CN \
  --format markdown

# Triage from a local JSON file
gitlink-cli workflow +triage --from shortcuts/workflow/testdata/issue_bug.json --format json

# Triage by read-only GitLink fetch
gitlink-cli workflow +triage --owner Gitlink --repo gitlink-cli --state open --limit 5 --format table

# Health for a healthy repository
gitlink-cli workflow +health \
  --repository Gitlink/gitlink-cli \
  --open-issues 3 \
  --open-prs 1 \
  --has-readme \
  --has-license \
  --has-contributing \
  --agent-readiness-known \
  --agent-readiness-score 9 \
  --format table

# Health for a risky repository
gitlink-cli workflow +health \
  --repository demo/repo \
  --open-issues 60 \
  --stale-issues 25 \
  --open-prs 12 \
  --stale-prs 6 \
  --recent-activity-known \
  --recent-activity-days 120 \
  --release-known=false \
  --format json

# Health with Chinese markdown output
gitlink-cli workflow +health \
  --repository Gitlink/gitlink-cli \
  --open-issues 3 \
  --open-prs 1 \
  --has-readme \
  --has-license \
  --has-contributing \
  --lang zh-CN \
  --format markdown

# Health by read-only GitLink fetch
gitlink-cli workflow +health --owner Gitlink --repo gitlink-cli --stale-days 30 --format table

# PR review summary by read-only GitLink fetch
gitlink-cli workflow +pr-summary --owner Gitlink --repo gitlink-cli --number 1 --format markdown

# PR review summary from a local JSON file
gitlink-cli workflow +pr-summary --from shortcuts/workflow/testdata/pr_summary.json --format json

# Repository workflow report by read-only GitLink fetch
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format markdown

# Repository workflow report from a local JSON file
gitlink-cli workflow +repo-report --from shortcuts/workflow/testdata/repo_report.json --format json
```

Output formats:

- `json` for scripts and AI Agents
- `table` for terminal review
- `markdown` for Issue comments, PR comments, release notes, and competition write-ups

Safety:

- Current workflow commands use local analysis by default and can also read GitLink data in read-only fetch mode.
- They do not modify remote GitLink data.
- They do not depend on LLM APIs.
- `workflow +pr-summary` does not comment, approve, reject, or merge pull requests.
- `workflow +repo-report` aggregates health, issue triage, and PR review summary signals without remote writes.

### Raw API

For endpoints not covered by shortcuts, use the Raw API directly:

```bash
# GET request
gitlink-cli api GET /users/me

# POST request
gitlink-cli api POST /Gitlink/forgeplus/issues --body '{"subject":"test","description":"..."}'

# POST request with body from a file
gitlink-cli api POST /Gitlink/forgeplus/issues --body-file issue.json

# POST request with body from stdin
Get-Content issue.json | gitlink-cli api POST /Gitlink/forgeplus/issues --body-stdin

# With query parameters
gitlink-cli api GET /Gitlink/forgeplus/commits --query 'page=1&limit=5'
```

## Global Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `--owner` | Repository owner | `--owner Gitlink` |
| `--repo` | Repository name | `--repo forgeplus` |
| `--format` | Output format (json/table/yaml; workflow also supports markdown) | `--format json` |
| `--debug` | Enable debug output | `--debug` |

**Automatic context resolution:** When running inside a git repository, `--owner` and `--repo` are automatically resolved from `git remote origin`.

## Branch Conventions

gitlink-cli supports bidirectional code sync between GitHub and GitLink:

| Platform | Default Branch |
|----------|---------------|
| GitHub | `main` |
| GitLink | `master` |

**Push to GitLink from local:**

```bash
# Method 1: Use git command directly
git push gitlink main:master

# Method 2: Configure git remote
git config remote.gitlink.push refs/heads/main:refs/heads/master
git push gitlink
```

## AI Agent Skills

The `skills/` directory contains Agent Skill files for AI-automated GitLink operations.

See [skills/README.md](skills/README.md) for details.

| Skill | Description |
|-------|-------------|
| `gitlink-shared` | Authentication, global parameters, safety rules, API notes |
| `gitlink-repo` | Repository operations (create, view, delete, fork, etc.) |
| `gitlink-issue` | Issue operations (create, update, close, comment, etc.) |
| `gitlink-pr` | Pull request operations (create, merge, review, etc.) |
| `gitlink-member` | Repository member and invite link management |
| `gitlink-branch` | Branch management (create, delete, list, protect, unprotect) |
| `gitlink-release` | Release management (create, view, delete, etc.) |
| `gitlink-ci` | CI/CD operations (builds, logs, etc.) |
| `gitlink-pipeline` | Pipeline workflow operations (runs, logs, enable, disable, delete, etc.) |
| `gitlink-search` | Search (repositories, users, etc.) |
| `gitlink-org` | Organization management (members, teams, etc.) |
| `gitlink-user` | User management (profile info, etc.) |
| `gitlink-pm` | Project management (sprints, kanban, weekly reports, etc.) |
| `gitlink-workflow` | AI-powered workflows (issue triage, PR review, release notes, etc.) |

## Project Structure

```
gitlink-cli/
├── cmd/                      # Cobra command definitions
│   ├── root.go               # Root command + global flags
│   ├── auth/                 # Authentication commands
│   ├── api/                  # Raw API commands
│   ├── config/               # Configuration commands
│   └── cmdutil/              # Global utilities
├── internal/                 # Internal packages
│   ├── auth/                 # Login, token storage, transport
│   ├── client/               # HTTP client + pagination
│   ├── config/               # Config file management
│   ├── context/              # Git remote resolution
│   └── output/               # Envelope + formatter
├── shortcuts/                # Shortcut implementations
│   ├── common/               # Framework (types, runner)
│   ├── repo/                 # Repository shortcuts
│   ├── issue/                # Issue shortcuts
│   ├── pr/                   # PR shortcuts
│   ├── member/               # Repository member shortcuts
│   ├── branch/               # Branch shortcuts
│   ├── release/              # Release shortcuts
│   ├── org/                  # Organization shortcuts
│   ├── ci/                   # CI shortcuts
│   ├── pipeline/             # Pipeline shortcuts
│   ├── search/               # Search shortcuts
│   ├── user/                 # User shortcuts
│   └── register.go           # Registration entry point
├── skills/                   # AI Agent Skills
│   ├── README.md             # Skills guide
│   ├── gitlink-shared/       # Shared rules
│   ├── gitlink-repo/         # Repository skill
│   ├── gitlink-issue/        # Issue skill
│   ├── gitlink-pr/           # PR skill
│   ├── gitlink-pm/           # Project management skill
│   └── ...
├── doc/                      # Design documents
│   ├── Design.md
│   ├── CODE_SYNC_STRATEGY_FINAL.md
│   └── ...
├── main.go
├── Makefile
├── go.mod
└── README.md
```

## Documentation

- [Skills Guide](skills/README.md) — AI Agent Skills detailed documentation
- [Design Document](doc/design.md) — Architecture design and development plan

## FAQ

### Q: How do I use gitlink-cli in scripts?

Use the `GITLINK_TOKEN` environment variable + `--format json` for structured output:

```bash
export GITLINK_TOKEN="your-private-token"
gitlink-cli repo +list --format json | jq '.data.projects[] | .name'
```

### Q: How does automatic owner/repo resolution work?

When running inside a git repository, the CLI automatically resolves `--owner` and `--repo` from `git remote origin`:

```bash
cd ~/my-gitlink-project
gitlink-cli issue +list  # Automatically uses the current repository
```

### Q: What if my token expires?

Re-authenticate:

```bash
# Username/password login
gitlink-cli auth login

# Or use a private token (generate at GitLink web → Settings → Private Tokens)
gitlink-cli auth login --token
```

### Q: How do I use gitlink-cli in CI/CD or non-interactive environments (e.g. Trae sandbox)?

Set the `GITLINK_TOKEN` environment variable — no `auth login` needed:

```bash
export GITLINK_TOKEN="your-private-token"
gitlink-cli repo +list   # Ready to use
gitlink-cli auth status   # Shows "✓ Logged in via GITLINK_TOKEN environment variable"
```

Priority: `GITLINK_TOKEN` env var > keyring/file stored token. When the env var is not set, the original interactive login flow works as before.

### Q: What if npm installs successfully but `gitlink-cli` reports a missing binary?

Reinstall first:

```bash
npm install -g @gitlink-ai/cli
```

If the error persists, check whether the release page contains the asset for your platform,
for example `gitlink-cli_<version>_windows_amd64.zip` on Windows x64.
You can also download the binary manually from the release page or build from source with `go install .`.

### Q: Where are credentials stored on Windows?

gitlink-cli uses Windows Credential Manager for secure token storage. If Credential Manager is unavailable, it automatically falls back to file storage (`~/.config/gitlink-cli/credentials`).

### Q: Where can I find the full API reference?

See [skills/gitlink-shared/REFERENCE.md](skills/gitlink-shared/REFERENCE.md).

## License

[MulanPSL-2.0](https://license.coscl.org.cn/MulanPSL2)
