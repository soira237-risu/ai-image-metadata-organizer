# 로드맵 / Roadmap

최종 갱신: 2026-07-11

## 현재 방향 / Current direction

- CLI와 `internal/appcore`를 우선 개발한다.
- GUI는 공통 코어를 얇게 호출하는 Wails 독립 데스크톱 앱으로 함께 개선한다.
- CLI 기능을 위해 CLI와 GUI를 따로 구현하지 않는다.
- GUI는 밝고 따뜻한 디자인 시스템과 명시적 상태 계약을 유지한다.

## 완료된 기반 / Completed foundation

- PNG/WebP 재귀 스캔
- NovelAI/ComfyUI/generic 메타데이터 추출
- SQLite 인덱스와 unchanged skip
- `scan`, `show`, `search`, `tags`, `stats`, `export`, `move`
- JSON 출력과 사람이 읽는 CLI 출력
- dry-run 기본 이동과 rollback
- 명령별 도움말과 엄격한 위치 인자·숫자 옵션 검증
- CLI와 GUI가 공유하는 `internal/appcore`
- `scan --fail-on-error`, Ctrl+C와 GUI 스캔 취소 전파
- 버전형 SQLite 마이그레이션, foreign key, busy timeout, batch hydration
- 원자적 JSON export와 볼륨 간 안전한 파일 이동
- 검색·상세 요청 경합 방지와 이동 계획 스냅샷 확인
- 밝고 따뜻한 3열 Wails GUI와 반응형 최소 창 레이아웃

## 다음 우선순위 / Next priorities

1. **오래된 인덱스 정리** — `scan --prune`과 안전한 삭제 계획을 추가한다.
2. **검색 성능 기준선** — 실제 대형 라이브러리로 batch hydration을 측정하고 FTS5 도입 여부를 결정한다.
3. **배포 단순화** — CLI와 GUI 실행 파일을 한 패키지로 만들고 재현 가능한 Windows 빌드 명령을 제공한다.
4. **키보드 흐름** — 결과 이동, 검색 초점, 상세 복사를 위한 단축키와 접근성 검증을 추가한다.
5. **인덱스 진단** — DB 버전, 마지막 스캔, 누락 파일을 읽기 전용으로 확인하는 진단 명령을 검토한다.

## GUI 범위 / GUI scope

GUI 작업 범위는 다음으로 제한한다.

- 폴더 선택, 진행 상태, 이미지 미리보기
- 검색·태그·통계 화면
- 이동 계획 확인과 명시적 적용
- export 저장 위치 선택
- 최근 요청 우선 상태 처리와 스캔 취소

스캔·검색·이동 규칙은 GUI에 다시 구현하지 않는다.

## 비목표 / Non-goals

- 클라우드 동기화 또는 계정 시스템
- 브라우저를 기본 사용자 인터페이스로 사용하는 로컬 웹앱
- CLI와 GUI의 별도 데이터 모델
- GUI 편의를 위해 CLI 안전 기본값을 약화하는 변경
- 측정 없이 대규모 프레임워크나 검색 엔진을 추가하는 작업

## 완료 정의 / Definition of done

- 동작이 공통 코어 또는 올바른 어댑터 경계에 구현됨
- 회귀 테스트가 있음
- 전체 Go 테스트, vet, CLI 빌드가 통과함
- 사용법이나 설계가 바뀌면 README·아키텍처·로드맵·ADR이 함께 갱신됨
