# Feishu Export Examples

## 1. Generate Workflow JSON

```bash
gitlink-cli workflow +repo-report --owner Gitlink --repo gitlink-cli --format json > report.json
```

## 2. Preview Notification Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --format json
```

## 3. Send Notification Card

```bash
gitlink-cli feishu +notify --from-workflow-json report.json --send --format table
```

## 4. Preview Wiki Export

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --format markdown
```

## 5. Export To Wiki

```bash
gitlink-cli feishu +doc-export \
  --from-workflow-json report.json \
  --wiki-url "https://example.feishu.cn/wiki/..." \
  --send \
  --format table
```

## 6. Generate Bitable Records

```bash
gitlink-cli feishu +bitable-records --from-workflow-json report.json --format json
```

