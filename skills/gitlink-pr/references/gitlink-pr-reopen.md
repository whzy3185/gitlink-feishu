# pr +reopen

Reopen a closed Pull Request.

## Usage

```bash
gitlink-cli pr +reopen --id 3
gitlink-cli pr +reopen -i 3
```

## Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `--id` / `-i` | Yes | PR number from the web URL `/pulls/N` |
| `--owner` | No | Repository owner, auto-detected from git remote when omitted |
| `--repo` | No | Repository name, auto-detected from git remote when omitted |
| `--format` | No | Output format: `json`, `table`, or `yaml` |

## API

```text
POST /v1/{owner}/{repo}/pulls/{number}/reopen
```

## Notes

- Use `pr +view -i <id>` first to confirm the PR is currently closed.
- This command uses the PR number shown in the web URL, not the internal database ID.
