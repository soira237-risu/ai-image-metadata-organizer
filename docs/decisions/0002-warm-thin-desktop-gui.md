# ADR 0002: 밝고 따뜻한 얇은 데스크톱 GUI / Warm thin desktop GUI

- 상태 / Status: Accepted
- 날짜 / Date: 2026-07-11

## 맥락 / Context

CLI와 GUI를 별도로 구현하면 같은 스캔·검색·이동 규칙이 갈라진다. 기존 React 화면은 단일 컴포넌트가 데이터 수명주기와 표시를 모두 담당해 오래된 비동기 응답, 폴더 전환 상태, 이동 계획 확인을 안전하게 다루기 어려웠다.

Building the CLI and GUI separately would split scan, search, and move rules. The previous single-component React screen mixed data lifecycles with rendering, making stale async responses, folder changes, and reviewed move plans difficult to handle safely.

## 결정 / Decision

- GUI는 Wails 독립 데스크톱 앱으로 유지한다.
- 제품 동작은 `internal/appcore`에 두고 Wails 백엔드는 얇은 어댑터로 유지한다.
- React는 컨트롤러 훅과 표시 컴포넌트로 분리한다.
- 시각 언어는 warm ivory, coral, sage의 밝고 따뜻한 팔레트와 3열 목록 레이아웃을 사용한다.
- 이동은 dry-run 계획, 요청 스냅샷, 앱 내부 확인을 거쳐야 한다.
- 브라우저 호스팅 UI, 별도 GUI 데이터 모델, 썸네일 그리드 중심 제품은 채택하지 않는다.

## 결과 / Consequences

- CLI 기능은 GUI 수정 없이 공통 코어에서 발전할 수 있다.
- GUI는 데스크톱 편의와 상태 관리에만 집중한다.
- Wails 계약과 React 상태 수명주기에 대한 회귀 테스트가 필요하다.
- 프런트 변경 시 임베드 산출물을 다시 생성하고 두 기준 창 크기에서 시각 확인한다.
