# AI Coding Agent Guide

이 문서는 AI 코딩 에이전트가 이 저장소를 수정할 때 지켜야 할 최소 규칙입니다.

This guide defines the minimum rules for AI coding agents working in this repository.

## 패키지 지도 / Package Map

- `cmd/imv`: CLI 진입점과 명령 dispatch
- `cmd/imv-gui`: Wails 데스크톱 진입점과 백엔드
- `internal/appcore`: CLI와 GUI가 공유하는 애플리케이션 서비스
- `internal/metadata`: NovelAI, ComfyUI, 일반 PNG/WebP 메타데이터 추출
- `internal/scanner`: PNG/WebP 파일 검색과 인덱싱
- `internal/store`: SQLite 스키마와 인덱스 저장소
- `internal/mover`: 이동 계획, 충돌 정책, 적용, 이동 로그
- `gui`: React + TypeScript + Vite 프런트엔드 소스
- `cmd/imv-gui/frontend/dist`: Wails 임베드용 생성 프런트엔드 결과물

## 필수 검증 / Required Verification

저장소 루트에서 다음 명령을 실행합니다.

Run these commands from the repository root:

```powershell
go version
go test ./...
go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe help
```

프런트엔드 변경이 있으면 `gui`에서 다음을 실행하고, 결과물 변경을 커밋합니다.

For frontend changes, run these commands from `gui` and commit the generated result:

```powershell
Set-Location .\gui
npm install
npm run test:run
npm run build
Set-Location ..
```

Wails 통합 변경은 `cmd/imv-gui`에서 확인합니다.

For Wails integration changes, verify from `cmd/imv-gui`:

```powershell
Set-Location .\cmd\imv-gui
wails doctor
wails build -m -nopackage -tags native_webview2loader -o imv-gui.exe
```

문서만 변경한 경우에도 `git diff --check`와 문서 링크/명령 일관성 검사를 실행합니다.

For documentation-only changes, still run `git diff --check` and documentation link/command consistency checks.

## 이동 안전성 / Move-Safety Invariants

- CLI `move`는 `--apply` 없이는 계획만 생성해야 합니다.
- GUI는 `PlanMove` 결과를 보여준 뒤 명시적인 확인을 거쳐 `ApplyMove`를 호출해야 합니다.
- source, destination, tag를 확인하지 않은 상태에서 실제 파일 이동을 실행하지 않습니다.
- 충돌 정책은 `skip` 또는 `rename`만 허용하며, 대상 경로는 태그를 안전한 경로 세그먼트로 정규화해야 합니다.
- 파일 이동과 DB 경로 갱신이 실패하면 가능한 경우 원래 경로로 rollback하고, 결과를 이동 로그에 남깁니다.
- CLI와 GUI의 이동 동작은 `internal/appcore`와 `internal/mover`의 공통 규칙을 유지해야 합니다.

## 변경 경계 / Change Boundaries

- 개인 이미지, prompt, SQLite DB, 로컬 경로를 커밋하거나 이슈/로그에 포함하지 않습니다.
- 실행 파일(`*.exe`)을 커밋하지 않습니다.
- UI를 변경하면 `gui` 프런트엔드를 다시 빌드하고 `cmd/imv-gui/frontend/dist` 변경을 함께 커밋합니다.
- CLI와 GUI가 공유하는 `internal/appcore` 동작을 보존합니다. 한쪽만 동작을 바꾸지 말고 공용 서비스와 양쪽 테스트를 확인합니다.
- 기존 변경을 되돌리지 않습니다. 작업 범위를 작은 focused change로 유지합니다.
- 테스트는 변경한 패키지와 사용자 흐름에 맞춘 focused tests를 우선 추가하고, 마지막에는 전체 검증을 실행합니다.
