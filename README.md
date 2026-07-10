# AI Image Metadata Organizer (IMV)

NovelAI 이미지를 먼저 지원하는 로컬 이미지 메타데이터 정리 도구입니다. PNG/WebP에서 prompt, negative prompt, 설정값과 태그를 읽고 검색하며, 확인한 계획에 따라 안전하게 파일을 정리합니다. ComfyUI workflow 요약도 지원합니다.

NovelAI-first local image metadata organizer for prompt/tag search, ComfyUI workflow summaries, and safe PNG/WebP file organization.

![IMV workflow: open image, read metadata, sort tags, confirm the move](assets/imv-illustrations/01-safe-metadata-flow.png)

`이미지 열기 -> 메타데이터 읽기 -> 태그 검색 -> 이동 계획 -> 확인 후 이동`

## 바로 시작하기

필요한 것: Windows PowerShell, Go 1.22+, Node `^20.19.0 || >=22.12.0`, Wails v2.

```powershell
git clone https://github.com/soira237-risu/ai-image-metadata-organizer.git
Set-Location .\ai-image-metadata-organizer

go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe scan "C:\Path\To\NovelAI-Images" --db .\.imv\imv.db
.\bin\imv.exe search --tag "blue hair" --db .\.imv\imv.db
```

GUI를 빌드하고 실행하려면:

```powershell
Set-Location .\cmd\imv-gui
wails build -m -nopackage -tags native_webview2loader -o imv-gui.exe
.\build\bin\imv-gui.exe
```

## 할 수 있는 일

- NovelAI prompt, negative prompt, 설정값, 태그 추출
- ComfyUI workflow 요약과 일반 PNG/WebP 메타데이터 확인
- 폴더 스캔, 태그/텍스트 검색, 통계, JSON 내보내기
- CLI 자동화와 Wails 데스크톱 GUI
- 태그를 기준으로 한 폴더 생성과 파일 이동

> 파일 이동은 안전하게 시작합니다. CLI `move`는 기본 dry-run이며, 실제 이동에는 `--apply`가 필요합니다. GUI도 이동 계획을 확인한 뒤에만 실행합니다.

<details>
<summary><strong>상세 설치와 사용법</strong></summary>

### CLI 검증과 빌드

저장소 루트에서 실행합니다.

```powershell
go mod download
go test ./...
go build -o .\bin\imv.exe .\cmd\imv
.\bin\imv.exe help
```

### GUI 개발과 빌드

Wails가 아직 없다면 먼저 설치하고, 현재 PowerShell 세션의 PATH에 추가합니다.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
$env:Path += ";$(go env GOPATH)\bin"
```

Wails 빌드는 프런트엔드 의존성 설치와 빌드를 함께 수행합니다. 프런트엔드만 확인할 때는 아래 명령을 사용합니다.

```powershell
Set-Location .\gui
npm install
npm run test:run
npm run build
Set-Location ..\cmd\imv-gui
wails doctor
wails build -m -nopackage -tags native_webview2loader -o imv-gui.exe
.\build\bin\imv-gui.exe
```

개발 중에는 `cmd\imv-gui`에서 `wails dev`를 실행합니다.

### CLI 예시

```powershell
$images = "C:\Path\To\NovelAI-Images"
.\bin\imv.exe scan $images --db .\.imv\imv.db
.\bin\imv.exe search --tag "1girl" --db .\.imv\imv.db
.\bin\imv.exe show 1 --raw --db .\.imv\imv.db
.\bin\imv.exe export --out .\imv-export.json --pretty --db .\.imv\imv.db
```

이동은 먼저 계획만 출력합니다. source와 destination을 확인한 뒤에만 `--apply`를 추가하세요.

```powershell
.\bin\imv.exe move --tag "1girl" --to "C:\Path\To\Organized" --db .\.imv\imv.db
.\bin\imv.exe move --tag "1girl" --to "C:\Path\To\Organized" --conflict rename --apply --db .\.imv\imv.db
```

전체 명령과 옵션은 다음으로 확인합니다.

```powershell
.\bin\imv.exe help
```

### GUI 사용

`파일 열기`는 PNG/WebP 한 장을 바로 확인합니다. `폴더 열기`는 폴더를 스캔해 검색, 태그 요약, 통계, JSON 내보내기를 제공합니다.

파일 이동은 `이동 태그`와 `대상 폴더`를 입력한 뒤 `계획`을 만들고, 결과를 확인한 후 `이동 실행`을 눌러야 합니다.

</details>

## 기여와 보안

[CONTRIBUTING.md](CONTRIBUTING.md)에서 기여 방법을, [AGENTS.md](AGENTS.md)에서 AI 작업 규칙을, [SECURITY.md](SECURITY.md)에서 비공개 보안 신고 방법을 확인하세요.

MIT License. No executables or private AI artwork are distributed with this repository.
