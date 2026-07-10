# 기여 안내 / Contributing

이 프로젝트는 초보 기여자와 AI의 도움을 받은 기여를 환영합니다. AI-assisted contributions are allowed, but every contribution still requires human review and tests.

## 시작하기 / Before You Start

1. 이슈를 먼저 읽고, 큰 변경은 작업 방향을 이슈에서 확인합니다.
2. 저장소를 clone하고 Go 1.22+, Node `^20.19.0 || >=22.12.0`, Wails v2를 준비합니다.
3. 개인 이미지, prompt, DB, 로컬 경로는 커밋하지 않습니다.

Read the issue first, prepare the required tools, and keep private images, prompts, databases, and local paths out of commits.

## 브랜치 / Branch

기능이나 수정마다 별도 브랜치를 만드세요. 브랜치 이름은 `feat/<short-name>`, `fix/<short-name>`, `docs/<short-name>`처럼 목적을 드러내면 좋습니다.

Use a separate branch for each change. Prefer names such as `feat/<short-name>`, `fix/<short-name>`, or `docs/<short-name>`.

```powershell
git switch -c docs/update-guide
```

## 구현과 테스트 / Implement and Test

변경 범위를 작게 유지하고 기존 CLI/GUI 공용 동작을 보존합니다. CLI와 GUI의 이동 기능은 먼저 계획을 확인하고, 실제 적용 경로는 focused tests로 검증합니다.

Keep the change focused and preserve shared CLI/GUI behavior. Verify move planning before applying moves, and cover the changed behavior with focused tests.

```powershell
go test ./...
Set-Location .\gui
npm install
npm run test:run
npm run build
Set-Location ..
git diff --check
```

UI를 바꿨다면 `cmd/imv-gui/frontend/dist`를 다시 빌드한 결과도 커밋해야 합니다. 실행 파일은 커밋하지 않습니다.

When changing the UI, commit the rebuilt `cmd/imv-gui/frontend/dist` output. Do not commit executables.

## 커밋 / Commit

커밋 메시지는 짧고 명령형으로 작성하며 한 커밋에는 하나의 논리적 변경만 담습니다. 예: `docs: clarify move safety`.

Use a short imperative commit subject and keep one logical change per commit. Example: `docs: clarify move safety`.

```powershell
git status --short
git add README.md AGENTS.md CONTRIBUTING.md SECURITY.md LICENSE
git commit -m "docs: clarify move safety"
```

## Pull Request

PR 설명에 변경 목적, 검증한 명령, 사용자 영향, 남은 우려를 적습니다. AI-assisted contributions는 사용한 도구나 생성 범위를 밝히고, 작성자가 직접 결과를 검토했다는 점을 포함하세요.

In the PR description, include the purpose, verification commands, user impact, and remaining concerns. Disclose AI assistance and state that a contributor reviewed the generated result.

리뷰 피드백을 반영한 뒤 테스트를 다시 실행하고, 병합 전까지 작업 브랜치를 최신 기준으로 유지합니다.

After addressing review feedback, rerun the tests and keep the branch current until merge.
