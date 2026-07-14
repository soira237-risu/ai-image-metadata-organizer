# ADR 0001: CLI 우선과 공통 코어 / CLI-first with a shared core

- 상태 / Status: Accepted
- 날짜 / Date: 2026-07-11

> 2026-07-11 갱신: 공통 코어 안정화 조건이 충족되어 GUI 유지보수 모드는 종료했다. CLI 우선·공통 코어 결정은 유지하며, 활성 GUI 범위와 상태·시각 계약은 ADR 0002가 보완한다.
>
> 2026-07-11 update: the shared-core stabilization condition has been met, so GUI maintenance mode has ended. The CLI-first shared-core decision remains; ADR 0002 supplements it with the active GUI scope and state/visual contract.

## 배경 / Context

CLI와 GUI를 동시에 기능 개발하면 플래그, 검증, 데이터 타입, 오류 처리, 스캔·검색·이동 규칙이 두 경로에서 어긋날 수 있다. 이 프로젝트는 개인용 로컬 도구로서 설치와 실행이 단순해야 하며, GUI가 필요할 때도 브라우저가 아닌 독립 데스크톱 앱을 선호한다.

Developing CLI and GUI features independently risks divergent flags, validation, data types, error handling, and scan/search/move rules. This personal local tool should remain simple to run, while any future GUI should be a standalone desktop application rather than a browser-first UI.

## 결정 / Decision

1. CLI를 제품의 기본 인터페이스이자 자동화 계약으로 삼는다.
2. 실제 기능은 `internal/appcore`와 하위 도메인 패키지에 한 번만 구현한다.
3. `cmd/imv`는 CLI 어댑터, `cmd/imv-gui`와 `gui`는 Wails 데스크톱 어댑터로 유지한다.
4. 공통 코어가 안정될 때까지 GUI는 유지보수 모드로 둔다.
5. 단일 실행 파일보다 단일 기능 소스를 우선한다. 배포 편의는 한 패키지와 한 빌드 명령으로 해결한다.

## 결과 / Consequences

### 장점 / Benefits

- 기능과 버그 수정을 한 곳에서 수행한다.
- CLI 테스트가 GUI가 사용할 핵심 동작도 검증한다.
- GUI 작업을 중단해도 CLI 개발이 막히지 않는다.
- 나중에 GUI를 다시 만들 때 파일 대화상자와 화면 흐름에 집중할 수 있다.

### 비용 / Costs

- CLI와 GUI 실행 파일은 분리될 수 있다.
- `appcore` API 변경 시 두 어댑터의 컴파일 계약을 확인해야 한다.
- 화면 상태와 파일 대화상자처럼 GUI 고유 코드는 별도로 유지해야 한다.

## 검토한 대안 / Alternatives considered

### CLI와 GUI에 기능을 각각 구현

중복과 동작 불일치 위험이 커서 채택하지 않는다.

### GUI가 CLI 프로세스와 JSON 출력을 직접 호출

언어에 독립적이지만 프로세스 수명, 취소, 오류 코드, 스트리밍 진행 상태 프로토콜을 별도로 관리해야 한다. 현재 Go/Wails 구성에서는 `appcore` 직접 호출보다 복잡하므로 기본 경로로 채택하지 않는다.

### `imv ui` 브라우저 인터페이스

단일 실행 파일 구성은 쉽지만 독립 데스크톱 GUI 선호와 맞지 않아 기본 경로에서 제외한다.

## 재검토 조건 / Revisit when

- GUI가 Go 외의 기술로 완전히 교체됨
- 외부 프로그램이 안정적인 원격 API를 요구함
- 단일 실행 파일이 실제 배포의 필수 조건이 됨
