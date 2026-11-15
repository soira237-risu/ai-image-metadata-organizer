# AGENTS.md

## 프로젝트 목표 / Project mission

`imv`는 NovelAI와 ComfyUI 이미지의 메타데이터를 로컬에서 색인·검색·내보내기·정리하는 CLI 우선 도구다. 현재 개발의 중심은 설치 부담이 낮은 Go CLI와 공통 코어이며, GUI는 같은 코어를 사용하는 얇은 독립 데스크톱 어댑터로 유지한다.

`imv` is a CLI-first local tool for indexing, searching, exporting, and organizing NovelAI and ComfyUI image metadata. Development focuses on the low-friction Go CLI and shared core; the GUI remains a thin standalone desktop adapter over that same core.

## 작업별 참고 문서 / Task-specific references

현재 작업에 필요한 문서만 참고한다. / Read the references relevant to the current change.

- 사용자 동작·CLI 계약을 변경할 때: `README.md`
- 코드 경계·데이터 흐름을 변경할 때: `docs/ARCHITECTURE.md`
- 다음 작업의 우선순위·비목표를 판단할 때: `docs/ROADMAP.md`
- 개발 환경·검증 명령을 확인할 때: `docs/DEVELOPMENT.md`
- 기존 기술 결정을 바꿀 때: `docs/decisions/`의 관련 ADR

## 변경 불변조건 / Change invariants

- 실제 제품 동작은 `internal/appcore`와 그 하위 패키지에 둔다.
- `cmd/imv`는 인자 해석, 입력 검증, 텍스트/JSON 출력만 담당한다.
- `cmd/imv-gui`와 `gui`에는 스캔·검색·이동 규칙을 중복 구현하지 않는다.
- CLI 기능 개발에 GUI 변경을 강제하지 않는다. 공통 API가 바뀐 경우에만 GUI 어댑터의 컴파일과 계약을 맞춘다.
- GUI는 브라우저 우선 제품으로 전환하지 않는다. Wails 기반 독립 데스크톱 앱을 기본으로 한다.
- 파일 이동은 dry-run이 기본이며 실제 쓰기는 명시적 `--apply` 뒤에만 수행한다.
- GUI 이동은 계획 검토와 명시적 확인 뒤에만 적용한다.
- 충돌 정책은 `skip` 또는 `rename`만 허용하며 실패 시 가능한 범위에서 원래 상태로 롤백한다.
- SQLite 단일 연결 구조에서는 열린 rows를 닫기 전에 상세 재조회를 하지 않는다.
- 새 기능과 버그 수정은 변경된 동작과 회귀 위험을 검증한다. 기존 테스트로 충분하지 않으면 관련 테스트를 추가한다.
- 개인 이미지, prompt, SQLite DB와 로컬 경로를 커밋하거나 이슈·로그에 포함하지 않는다.

## 작업 순서 / Working sequence

1. `git status --short`로 기존 사용자 변경을 확인하고 보존한다.
2. 변경이 CLI 표현인지 공통 동작인지 먼저 분류한다.
3. 공통 동작 변경은 `internal/appcore` 또는 해당 하위 패키지의 관련 테스트로 검증한다.
4. 최소 구현 후 CLI 어댑터를 연결한다.
5. 공통 API가 바뀌었으면 Wails 백엔드가 계속 컴파일되는지 확인한다.
6. 프런트엔드를 수정했으면 `cmd/imv-gui/frontend/dist`를 다시 생성해 함께 커밋한다.
7. 사용자 동작이나 설계 결정이 달라졌으면 관련 문서와 ADR을 함께 갱신한다.

## 변경 범위별 검증 / Verification by change scope

Go 코드 또는 Go 빌드 설정을 변경했다면 아래 기본 검증을 실행한다.

Windows에서는 전역 Go가 없으면 `.tools/go-portable-1.26.4/go/bin/go.exe`를 사용하고 `GOMODCACHE`와 `GOCACHE`를 작업공간 내부로 지정한다.

```powershell
go test ./...
go vet ./...
go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe help
```

GUI 또는 공통 API를 수정했다면 `go build ./cmd/imv-gui`도 실행한다. `gui`나 임베드 산출물을 수정한 경우 다음 검증을 추가한다.

```powershell
Set-Location .\gui
npm ci
npm run typecheck
npm run test:run
npm run build
Set-Location ..
```

문서만 수정한 경우에는 `git diff --check`와 변경한 링크·명령의 일관성을 확인한다. 동작이나 빌드 설정이 바뀌지 않았다면 전체 테스트·빌드를 반복하지 않는다. 관련 검증이 통과한 뒤에는 새로운 변경·실패·미해결 우려가 있을 때만 범위를 넓히거나 재실행한다.

## 문서 규칙 / Documentation rules

- `README.md`는 한글/영어 병기 형태를 유지한다.
- 구조 변경은 `docs/ARCHITECTURE.md`, 우선순위 변경은 `docs/ROADMAP.md`에 반영한다.
- 장기간 유지할 결정은 `docs/decisions/NNNN-*.md` ADR로 남긴다.
- 현재 환경의 테스트·브랜치·생성 산출물 상태는 과거 기록을 재사용하지 말고 매번 다시 확인한다.
