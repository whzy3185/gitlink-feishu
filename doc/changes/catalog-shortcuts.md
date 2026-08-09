# Catalog Shortcuts

Adds a `catalog` shortcut group for GitLink platform template lookups:

- `catalog +licenses` lists repository license templates.
- `catalog +ignores` lists repository `.gitignore` templates.
- Both commands support `--name` filtering and return the original API response in the configured output format.

This closes a small but useful OpenAPI coverage gap for repository bootstrap workflows. Agents and scripts can now discover valid license and ignore template names before creating repositories, without falling back to raw API paths.
