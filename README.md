# imv

`imv` is a local-first image metadata viewer engine for AI images.

## Scope

- PNG and WebP scanning
- NovelAI and ComfyUI metadata extraction
- SQLite index at `.imv/imv.db` by default
- Search, tag summaries, index stats, JSON export, and tag-based file organization
- Safe move behavior: dry-run by default, `--apply` required for writes
- Split executables: `imv` for CLI automation and `imv-gui` for Wails desktop use

## Structure

- `cmd/imv`: CLI entry point
- `cmd/imv-gui`: Wails backend and desktop entry point
- `internal/appcore`: shared scan/search/show/tags/stats/export/move service
- `gui`: React + TypeScript + Vite frontend source
- `cmd/imv-gui/frontend/dist`: generated frontend assets embedded into `imv-gui`

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

GUI development also requires Node/NPM and Wails:

```powershell
cd gui
npm install
npm run test:run
npm run build
cd ..
go test ./...
wails doctor
cd cmd/imv-gui
wails build -m -nopackage -tags native_webview2loader -o imv-gui.exe
wails dev
```

The GUI source lives under `gui`, while Vite builds into `cmd/imv-gui/frontend/dist` so the Wails command package can embed the production assets. Wails commands should be run from `cmd/imv-gui` because this repository keeps CLI and GUI executables split under `cmd/`.
