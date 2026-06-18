# imv

`imv`는 AI 이미지용 로컬 우선 이미지 메타데이터 뷰어 엔진입니다.

`imv` is a local-first image metadata viewer engine for AI images.

## 범위 / Scope

- PNG 및 WebP 스캔 / PNG and WebP scanning
- NovelAI 및 ComfyUI 메타데이터 추출 / NovelAI and ComfyUI metadata extraction
- 기본 SQLite 인덱스 경로: `.imv/imv.db` / SQLite index at `.imv/imv.db` by default
- 검색, 태그 요약, 인덱스 통계, JSON 내보내기, 태그 기반 파일 정리 / Search, tag summaries, index stats, JSON export, and tag-based file organization
- 안전한 이동 동작: 기본은 dry-run이며, 쓰기 작업에는 `--apply` 필요 / Safe move behavior: dry-run by default, `--apply` required for writes
- 실행 파일 분리: CLI 자동화용 `imv`, Wails 데스크톱용 `imv-gui` / Split executables: `imv` for CLI automation and `imv-gui` for Wails desktop use

## 구조 / Structure

- `cmd/imv`: CLI 진입점 / CLI entry point
- `cmd/imv-gui`: Wails 백엔드 및 데스크톱 진입점 / Wails backend and desktop entry point
- `internal/appcore`: 스캔, 검색, 표시, 태그, 통계, 내보내기, 이동 공용 서비스 / Shared scan/search/show/tags/stats/export/move service
- `gui`: React + TypeScript + Vite 프런트엔드 소스 / React + TypeScript + Vite frontend source
- `cmd/imv-gui/frontend/dist`: `imv-gui`에 임베드되는 생성된 프런트엔드 에셋 / Generated frontend assets embedded into `imv-gui`

## 명령 / Commands

```powershell
imv scan <folder> [--db .imv/imv.db] [--workers 4] [--rescan] [--json]
imv show <path-or-id> [--raw] [--json]
imv search [--tag <tag>] [--source nai|comfyui] [--format png|webp] [--has-workflow] [--q <text>] [--limit 50] [--long] [--json]
imv tags [--source nai|comfyui|generic|unknown] [--q <text>] [--limit 100] [--json]
imv stats [--json]
imv export --out <file.json> [--pretty]
imv move --tag <tag> --to <folder> [--apply] [--conflict skip|rename]
```

## GUI 사용 / GUI Usage

- `열기 > 파일 열기`: PNG/WebP 파일 하나만 즉시 읽어 미리보기와 메타데이터를 표시합니다. 이 동작은 폴더 전체를 스캔하지 않습니다.
- `열기 > 폴더 열기`: 폴더를 기준으로 스캔, 검색, 태그 요약, 통계를 사용합니다.
- 드래그 앤 드롭: 이미지 파일 하나를 창에 놓으면 단일 파일 모드로 바로 불러옵니다.
- `초기화`: 현재 화면 상태만 비우며 DB나 이미지 파일은 삭제하지 않습니다.
- 태그 기반 이동: `이동 태그`와 `대상 폴더`를 입력해 `계획`을 만든 뒤 `이동 실행`으로 실제 이동합니다. 실행 후 하단 결과 패널에서 이동 상태, 이미지 미리보기, `이동 폴더 열기`를 확인할 수 있습니다.
- `옵션`: 앱 내부 팝업으로 열리며 `화면`, `레이아웃`, `도움말` 탭을 제공합니다.
- `옵션 > 화면`: 글자 크기, 패널 간격, 미리보기 높이, 목록 밀도를 조절합니다.
- `옵션 > 레이아웃`: 왼쪽 필터 패널과 오른쪽 상세 패널 폭을 조절합니다. 작업공간의 얇은 분할 손잡이를 드래그해도 폭을 바꿀 수 있습니다.
- `옵션 > 도움말`: 파일 열기, 폴더 열기, 드래그 앤 드롭, 이동 실행 흐름을 확인합니다.
- 하단 상태바: 파일 불러오기와 스캔 진행 상황을 `0/1`, `N/M` 형태로 표시합니다.

## 개발 / Development

빌드와 테스트에는 Go가 필요합니다.

Go is required to build and test:

```powershell
go mod download
go test ./...
go build ./cmd/imv
```

GUI 개발에는 Node/NPM과 Wails도 필요합니다.

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

GUI 소스는 `gui` 아래에 있고, Vite 빌드 결과는 `cmd/imv-gui/frontend/dist`에 생성됩니다. 이렇게 해야 Wails 명령 패키지가 프로덕션 에셋을 임베드할 수 있습니다. 이 저장소는 CLI와 GUI 실행 파일을 `cmd/` 아래에서 분리하므로 Wails 명령은 `cmd/imv-gui`에서 실행해야 합니다.

The GUI source lives under `gui`, while Vite builds into `cmd/imv-gui/frontend/dist` so the Wails command package can embed the production assets. Wails commands should be run from `cmd/imv-gui` because this repository keeps CLI and GUI executables split under `cmd/`.
