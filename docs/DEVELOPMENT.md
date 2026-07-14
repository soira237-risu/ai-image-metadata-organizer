# 개발과 검증 / Development and verification

## 도구 선택 / Tool selection

전역 `go`가 있으면 사용한다. 없으면 저장소에 준비된 portable Go를 사용한다.

```powershell
$go = '.\.tools\go-portable-1.26.4\go\bin\go.exe'
$env:GOMODCACHE = Join-Path (Get-Location) '.gomodcache'
$env:GOCACHE = Join-Path (Get-Location) '.gocache'
& $go version
```

작업공간 내부 캐시를 먼저 지정해야 Windows 샌드박스가 기본 사용자 캐시 접근을 막는 상황을 피할 수 있다.

## CLI 개발 루프 / CLI development loop

```powershell
& $go test ./...
& $go vet ./...
& $go build -o '.\bin\imv.exe' '.\cmd\imv'
& '.\bin\imv.exe' help
```

커버리지 확인:

```powershell
& $go test -cover ./...
```

CLI 옵션이나 출력 계약을 수정했다면 `cmd/imv/main_test.go`에서 다음을 확인한다.

- 플래그가 위치 인자 앞뒤에서 동일하게 동작하는가
- 잘못된 입력이 DB나 파일 작업 전에 거부되는가
- `--help`가 stdout과 종료 코드 0을 사용하는가
- `--json` 출력이 유효하고 기존 필드와 호환되는가
- 실제 오류가 비정상 종료 코드로 전달되는가

## 공통 코어 변경 / Shared-core changes

공통 기능은 CLI 함수에 바로 넣지 말고 `internal/appcore` 또는 적절한 하위 패키지에 추가한다. 다음 순서를 사용한다.

1. 원하는 공통 API를 테스트로 표현한다.
2. 실패 원인을 확인한다.
3. 최소 구현을 추가한다.
4. 관련 패키지와 전체 테스트를 실행한다.
5. CLI 어댑터를 연결한다.
6. 공통 API가 바뀌었으면 GUI 백엔드 빌드를 확인한다.

## GUI를 수정해야 하는 경우 / When GUI work is required

CLI 기능만 추가할 때는 GUI를 수정하지 않는다. 다음 경우에만 GUI 검증을 수행한다.

- `appcore` 요청·응답 타입이 변경됨
- Wails 백엔드 메서드가 변경됨
- React 화면 또는 임베드 자산이 변경됨

Go 백엔드 확인:

```powershell
& $go build -o '.\bin\imv-gui.exe' '.\cmd\imv-gui'
```

프론트엔드까지 변경했다면:

```powershell
Set-Location '.\gui'
npm ci
npm run typecheck
npm run test:run
npm run build
Set-Location '..'
```

생성된 `cmd/imv-gui/frontend/dist`는 줄바꿈 차이로 CI diff가 생길 수 있으므로 변경 내용을 반드시 확인한다.

GUI 상태 변경은 다음 회귀 계약을 우선 확인한다.

- 오래된 검색·상세 요청이 최신 선택을 덮어쓰지 않는가
- 폴더 전환 시 이전 레코드·상세·통계가 남지 않는가
- Wails 이벤트 구독이 화면 해제 시 정리되는가
- 이동 계획 뒤 입력 변경이 적용 버튼을 비활성화하는가
- 적용 요청이 사용자가 검토한 계획 요청과 정확히 같은가
- 스캔 중 취소 버튼이 백엔드 context를 취소하는가

## 완료 체크리스트 / Completion checklist

- 기존 사용자 변경과 미추적 파일을 보존했는가
- 새 동작에 회귀 테스트가 있는가
- 전체 Go 테스트와 vet이 통과하는가
- CLI 빌드와 관련 도움말을 실제 실행했는가
- GUI와 무관한 작업이 GUI 소스나 생성 자산을 건드리지 않았는가
- 구조·우선순위·결정 변경을 관련 문서에 반영했는가
- GUI 변경이면 1280×820과 960×640에서 잘림·겹침을 시각 확인했는가
