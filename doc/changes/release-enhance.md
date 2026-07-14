# Release Enhance Shortcuts

Submitter: Wang Yue

This change adds two new release shortcuts for getting the latest release and auto-generating release notes.

## Commands

- Add `release +latest` for getting the latest release version.
- Add `release +auto-notes` for auto-generating release notes from commits and issues.

## Behavior

### release +latest

- Fetches releases from the repository and returns the first matching release.
- By default, skips draft and prerelease versions.
- Supports `--include-prerelease` flag to include prerelease versions.
- Supports `--include-draft` flag to include draft versions.
- Returns error if no matching release is found.

### release +auto-notes

- Generates formatted release notes from commit messages and closed issues.
- Automatically categorizes commits by prefix:
  - `feat:` → 🚀 新功能 (New Features)
  - `fix:` → 🐛 Bug 修复 (Bug Fixes)
  - Others → 📝 其他变更 (Other Changes)
- Supports `--from-tag` to specify the starting tag for comparison.
- If `--from-tag` is not specified, uses the last 50 commits.
- Includes closed issues in the "Related Issues" section.
- Supports `--format json` to output with statistics (commits_count, issues_count).

## Verification

- Unit tests cover:
  - `TestReleaseLatest`: Basic latest release retrieval
  - `TestReleaseLatestWithPrerelease`: Including prerelease versions
  - `TestReleaseLatestSkipsDraft`: Skipping draft versions
  - `TestReleaseLatestNoReleases`: Error handling when no releases
  - `TestReleaseAutoNotes`: Basic auto-notes generation
  - `TestReleaseAutoNotesWithFromTag`: Using from-tag parameter
  - `TestReleaseAutoNotesJSONFormat`: JSON format output with statistics
