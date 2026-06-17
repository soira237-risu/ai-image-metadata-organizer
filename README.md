# imv

`imv` is a local-first CLI engine for indexing AI image metadata.

## Scope

- PNG and WebP scanning
- NovelAI and ComfyUI metadata extraction
- SQLite index at `.imv/imv.db` by default
- Search, tag summaries, index stats, JSON export, and tag-based file organization
- Safe move behavior: dry-run by default, `--apply` required for writes

## Commands

```powershell
imv scan <folder> [--db .imv/imv.db] [--workers 4] [--rescan] [--json]
imv show <path-or-id> [--raw] [--json]
imv search [--tag <tag>] [--source nai|comfyui] [--format png|webp] [--has-workflow] [--q <text>] [--limit 50] [--long] [--json]
imv tags [--source nai|comfyui|generic|unknown] [--q <text>] [--limit 100] [--json]
imv stats [--json]
imv export --out <file.json> [--pretty]
imv move --tag <tag> --to <folder> [--apply] [--conflict skip|rename]
```

## Development

Go is required to build and test:

```powershell
go mod download
go test ./...
go build ./cmd/imv
```
