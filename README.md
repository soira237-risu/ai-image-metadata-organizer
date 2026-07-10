# AI Image Metadata Organizer (IMV)

NovelAI 이미지의 메타데이터를 로컬 우선 방식으로 살펴보고 prompt tag로 파일을 정리하는 NovelAI-first 도구입니다. ComfyUI workflow 요약과 일반적인 AI PNG/WebP 메타데이터도 함께 지원합니다.

This local-first tool inspects NovelAI image metadata and organizes files by prompt tags. It also supports ComfyUI workflow summaries and generic AI metadata in PNG/WebP files.

## 핵심 흐름 / Core Flow

`이미지 열기 -> 메타데이터 추출 -> 태그 검색 -> 이동 계획 -> 확인 후 이동`

Open an image, extract metadata, search tags, create a move plan, review it, and then apply the move.

## 주요 기능 / Features

- NovelAI prompt, negative prompt, 설정값, prompt tag 추출 / NovelAI prompt, negative prompt, settings, and prompt-tag extraction
- ComfyUI workflow 요약 및 일반 PNG/WebP 메타데이터 지원 / ComfyUI workflow summaries and generic PNG/WebP metadata
- PNG/WebP 파일을 재귀적으로 스캔하고 로컬 SQLite 인덱스에 저장 / Recursive PNG/WebP scanning into a local SQLite index
- CLI 자동화와 Wails 데스크톱 GUI 제공 / CLI automation and a Wails desktop GUI
- CLI `move`는 기본적으로 dry-run이며 `--apply`가 있어야 파일을 이동합니다 / CLI `move` is a dry-run by default and requires `--apply` to move files
- GUI는 이동 계획을 만든 뒤 명시적으로 확인해야 실제 이동을 실행합니다 / GUI requires a move plan and explicit confirmation before applying moves

## 프로젝트 메타데이터 / Project Metadata

GitHub description:

> NovelAI-first local image metadata organizer with prompt/tag search, ComfyUI support, and safe PNG/WebP file organization.

Repository: <https://github.com/soira237-risu/ai-image-metadata-organizer>

## 사전 요구사항 / Prerequisites

- Windows PowerShell
- Go 1.22+
- Node `^20.19.0 || >=22.12.0`
- Wails v2

이 저장소는 미리 빌드된 바이너리를 배포하지 않습니다. 소스 코드를 clone한 뒤 직접 빌드하세요.

No binaries are published. Clone the source repository and build locally.

## 설치와 빌드 / Install and Build

### 소스 받기 / Clone

```powershell
git clone https://github.com/soira237-risu/ai-image-metadata-organizer.git
Set-Location .\ai-image-metadata-organizer
```

### CLI 빌드 / Build the CLI

저장소 루트에서 실행합니다. `bin\imv.exe`가 생성됩니다.

Run from the repository root. This creates `bin\imv.exe`.

```powershell
go mod download
go test ./...
go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe help
```

### GUI 프런트엔드 / Build the frontend

프런트엔드 명령은 `gui` 디렉터리에서 실행합니다. `npm run build`는 Wails가 임베드할 `cmd/imv-gui/frontend/dist`를 갱신합니다.

Run frontend commands from `gui`. `npm run build` updates `cmd/imv-gui/frontend/dist`, which Wails embeds.

```powershell
Set-Location .\gui
npm install
npm run test:run
npm run build
Set-Location ..
```

### Wails 빌드 / Build with Wails

Wails 명령은 `cmd/imv-gui` 디렉터리에서 실행합니다. 프런트엔드를 먼저 빌드한 뒤 실행 파일을 만드세요.

Run Wails commands from `cmd/imv-gui`. Build the frontend first, then create the desktop executable.

```powershell
Set-Location .\cmd\imv-gui
wails doctor
wails build -m -nopackage -tags native_webview2loader -o imv-gui.exe
.\build\bin\imv-gui.exe
```

빌드가 끝나면 같은 `cmd/imv-gui` 디렉터리에서 `.\build\bin\imv-gui.exe`로 GUI를 실행합니다.

After the build finishes, launch the GUI from the same `cmd/imv-gui` directory with `.\build\bin\imv-gui.exe`.

개발 중에는 같은 디렉터리에서 `wails dev`를 사용할 수 있습니다.

During development, run `wails dev` from the same directory.

```powershell
wails dev
```

## CLI 사용 / CLI Usage

CLI는 저장소 루트에서 `.\bin\imv.exe`로 실행합니다. 먼저 이미지 폴더를 스캔하세요.

Run the CLI from the repository root as `.\bin\imv.exe`. Start by scanning an image folder.

```powershell
$images = "C:\Path\To\NovelAI-Images"
.\bin\imv.exe scan $images --db .\.imv\imv.db
.\bin\imv.exe search --tag "1girl" --db .\.imv\imv.db
.\bin\imv.exe show 1 --raw --db .\.imv\imv.db
.\bin\imv.exe export --out .\imv-export.json --pretty --db .\.imv\imv.db
```

`move`는 먼저 계획만 출력합니다. 출력된 source와 destination을 확인한 뒤에만 `--apply`를 추가하세요.

`move` prints a plan first. Review every source and destination, then add `--apply` only when the plan is correct.

```powershell
.\bin\imv.exe move --tag "1girl" --to "C:\Path\To\Organized" --db .\.imv\imv.db
.\bin\imv.exe move --tag "1girl" --to "C:\Path\To\Organized" --conflict rename --apply --db .\.imv\imv.db
```

## GUI 사용 / GUI Usage

GUI에서는 `파일 열기`로 PNG/WebP 하나를 바로 확인하거나 `폴더 열기`로 폴더를 스캔할 수 있습니다. 폴더를 열면 검색, 태그 요약, 통계, JSON 내보내기를 사용할 수 있습니다.

In the GUI, use `파일 열기` to inspect one PNG/WebP or `폴더 열기` to scan a folder. Folder mode provides search, tag summaries, statistics, and JSON export.

이동은 `이동 태그`와 `대상 폴더`를 입력해 `계획`을 만든 뒤, 결과를 확인하고 `이동 실행`을 명시적으로 눌러야 합니다. 계획 확인 전에는 파일이 이동되지 않습니다.

For a move, enter `이동 태그` and `대상 폴더`, create a `계획`, review its results, and explicitly press `이동 실행`. No files move before the plan is confirmed.

## 지원 명령 / Supported Commands

```text
imv scan <folder> [--db .imv/imv.db] [--workers 4] [--rescan] [--json]
imv show <path-or-id> [--db .imv/imv.db] [--raw] [--json]
imv search [--db .imv/imv.db] [--tag <tag>] [--source <source>] [--format png|webp] [--has-workflow] [--q <text>] [--limit 50] [--long] [--json]
imv tags [--db .imv/imv.db] [--source <source>] [--q <text>] [--limit 100] [--json]
imv stats [--db .imv/imv.db] [--json]
imv export [--db .imv/imv.db] --out <file.json> [--pretty]
imv move [--db .imv/imv.db] --tag <tag> --to <folder> [--apply] [--conflict skip|rename] [--json]
```

`source` 값은 메타데이터 종류에 따라 `nai`, `comfyui`, `generic`, `unknown`을 사용합니다. `format` 값은 `png` 또는 `webp`입니다.

Use `nai`, `comfyui`, `generic`, or `unknown` for `source`. Use `png` or `webp` for `format`.

## 구조 / Architecture

- `cmd/imv`: CLI 진입점 / CLI entry point
- `cmd/imv-gui`: Wails 백엔드와 데스크톱 진입점 / Wails backend and desktop entry point
- `internal/appcore`: CLI와 GUI가 공유하는 스캔, 검색, 표시, 태그, 통계, 내보내기, 이동 서비스 / Shared scan, search, show, tag, stats, export, and move services
- `internal/metadata`: NovelAI, ComfyUI, 일반 PNG/WebP 메타데이터 추출 / NovelAI, ComfyUI, and generic PNG/WebP extraction
- `internal/scanner`: 이미지 파일 스캔 / Image-file scanning
- `internal/store`: SQLite 인덱스와 `.imv/imv.db` 관리 / SQLite index and `.imv/imv.db` management
- `internal/mover`: 계획 생성, 충돌 처리, 적용, 이동 로그 / Planning, conflict handling, application, and move logs
- `gui`: React + TypeScript + Vite 소스 / React + TypeScript + Vite source
- `cmd/imv-gui/frontend/dist`: Wails에 임베드되는 생성된 프런트엔드 에셋 / Generated frontend assets embedded by Wails

## 기여와 보안 / Contributing and Security

기여 방법은 [CONTRIBUTING.md](CONTRIBUTING.md), AI 코딩 에이전트 지침은 [AGENTS.md](AGENTS.md), 보안 문제 신고는 [SECURITY.md](SECURITY.md)를 참고하세요. 라이선스는 [MIT](LICENSE)입니다.

See [CONTRIBUTING.md](CONTRIBUTING.md) for contributions, [AGENTS.md](AGENTS.md) for AI coding agents, and [SECURITY.md](SECURITY.md) for security reports. This project is licensed under the [MIT License](LICENSE).
