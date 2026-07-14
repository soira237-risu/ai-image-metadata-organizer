# imv

`imv`는 AI 이미지 메타데이터를 로컬에서 색인하고 정리하는 CLI 우선 도구입니다.

`imv` is a CLI-first, local-first metadata indexer and organizer for AI images.

## 다운로드 / Download

Windows 사용자는 [GitHub Releases](https://github.com/soira237-risu/ai-image-metadata-organizer/releases/latest)에서 최신 `imv-vX.Y.Z-windows-amd64.zip`을 받을 수 있습니다. 압축 파일에는 CLI `imv.exe`와 선택형 데스크톱 GUI `imv-gui.exe`가 함께 들어갑니다.

Windows users can download the latest `imv-vX.Y.Z-windows-amd64.zip` from [GitHub Releases](https://github.com/soira237-risu/ai-image-metadata-organizer/releases/latest). It includes the `imv.exe` CLI and the optional `imv-gui.exe` desktop app.

> 실행 파일은 아직 코드 서명되지 않아 Windows SmartScreen 경고가 표시될 수 있습니다. / The executables are not code-signed yet, so Windows SmartScreen may display a warning.

## 범위 / Scope

- PNG 및 WebP 스캔 / PNG and WebP scanning
- NovelAI 및 ComfyUI 메타데이터 추출 / NovelAI and ComfyUI metadata extraction
- 기본 `.imv/imv.db` SQLite 인덱스 / SQLite index at `.imv/imv.db` by default
- 검색, 태그 요약, 통계, JSON 내보내기, 태그 기반 파일 정리 / Search, tag summaries, statistics, JSON export, and tag-based file organization
- 안전한 이동: dry-run이 기본이며 실제 쓰기에는 `--apply` 필요 / Safe moves: dry-run by default, `--apply` required for writes

## 개발 방향 / Development direction

현재 개발은 CLI와 `internal/appcore`를 중심에 두고, GUI는 같은 공통 코어를 호출하는 얇은 Wails 독립 데스크톱 어댑터로 함께 개선합니다. CLI 기능을 GUI에 다시 구현하지 않습니다.

Current development keeps the CLI and `internal/appcore` at the center while improving the GUI as a thin standalone Wails desktop adapter over the same shared core. CLI behavior is not reimplemented in the GUI.

## 구조 / Structure

- `cmd/imv`: CLI 진입점 / CLI entry point
- `internal/appcore`: CLI와 GUI가 공유하는 애플리케이션 API / application API shared by CLI and GUI
- `internal/metadata`, `scanner`, `store`, `mover`: 핵심 도메인 패키지 / core domain packages
- `cmd/imv-gui`: 얇은 Wails 백엔드와 데스크톱 진입점 / thin Wails backend and desktop entry point
- `gui`: 밝고 따뜻한 React 데스크톱 프론트엔드 / bright, warm React desktop frontend
- `cmd/imv-gui/frontend/dist`: GUI에 임베드되는 생성 자산 / generated assets embedded into the GUI

## 명령 / Commands

```powershell
imv scan <folder> [--db .imv/imv.db] [--workers 4] [--rescan] [--fail-on-error] [--json]
imv show <path-or-id> [--raw] [--json]
imv search [--tag <tag>] [--source nai|comfyui] [--format png|webp] [--has-workflow] [--q <text>] [--limit 50] [--long] [--json]
imv tags [--source nai|comfyui|generic|unknown] [--q <text>] [--limit 100] [--json]
imv stats [--json]
imv export --out <file.json> [--pretty]
imv move --tag <tag> --to <folder> [--apply] [--conflict skip|rename]
```

명령별 옵션은 `imv help <command>`로 확인할 수 있습니다. Use `imv help <command>` for command-specific options.

`scan`은 기본적으로 파일별 오류를 결과에 포함하면서 나머지 파일을 계속 처리합니다. 자동화에서 하나의 파일 오류도 실패로 취급하려면 `--fail-on-error`를 사용하세요.

By default, `scan` reports per-file errors and continues processing the remaining files. Use `--fail-on-error` when automation should treat any file error as a failed command.

## 데스크톱 GUI / Desktop GUI

GUI는 폴더 선택, 스캔과 취소, 검색·태그·통계, 이미지 상세, JSON 내보내기, dry-run 이동 계획 확인을 제공합니다. 실제 이동은 앱 내부 확인 대화상자를 거쳐야 하며, 계획 후 대상·태그·충돌 설정이 바뀌면 새 계획을 만들어야 합니다.

The GUI provides folder selection, scan and cancellation, search, tags, statistics, image details, JSON export, and dry-run move-plan review. Applying a move requires an in-app confirmation, and changing the destination, tag, or conflict policy invalidates the reviewed plan.

```powershell
Set-Location .\gui
npm ci
npm run test:run
npm run build
Set-Location ..
go build -o .\bin\imv-gui.exe .\cmd\imv-gui
```

## 지속 개발 문서 / Continuous development documents

- [`AGENTS.md`](AGENTS.md): 에이전트 작업 규칙과 불변조건 / agent rules and invariants
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md): 구조와 코드 경계 / architecture and code boundaries
- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md): 개발·검증 절차 / development and verification
- [`docs/ROADMAP.md`](docs/ROADMAP.md): 우선순위와 비목표 / priorities and non-goals
- [`docs/GUI_DESIGN.md`](docs/GUI_DESIGN.md): GUI 상태·시각 계약 / GUI state and visual contract
- [`docs/RELEASING.md`](docs/RELEASING.md): GitHub 이름과 실행 파일 배포 절차 / GitHub identity and executable release process
- [`docs/decisions/0001-cli-first-shared-core.md`](docs/decisions/0001-cli-first-shared-core.md): CLI 우선 결정 기록 / CLI-first decision record
- [`docs/decisions/0002-warm-thin-desktop-gui.md`](docs/decisions/0002-warm-thin-desktop-gui.md): 데스크톱 GUI 결정 기록 / desktop GUI decision record

## 개발 / Development

Go가 필요합니다. 전역 Go가 없으면 저장소의 portable Go를 사용할 수 있습니다.

Go is required. The repository-local portable Go can be used when Go is not installed globally.

```powershell
go test ./...
go vet ./...
go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe help
```

자세한 Windows 캐시 설정과 GUI 조건부 검증은 [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)를 참고하세요.

See [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) for Windows-local cache setup and conditional GUI verification.
