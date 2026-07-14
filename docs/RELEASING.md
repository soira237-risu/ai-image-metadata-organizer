# 릴리스 / Releasing

## GitHub 프로젝트 이름 / GitHub project identity

저장소 이름은 제품명과 같은 `imv`를 권장한다.

The recommended repository name is `imv`, matching the product name.

- 저장소 / Repository: `soira237-risu/imv`
- 설명 / Description: `CLI-first local AI image metadata indexer and organizer for NovelAI and ComfyUI, with an optional Wails desktop GUI.`
- 추천 토픽 / Suggested topics: `imv`, `novelai`, `comfyui`, `ai-image`, `metadata`, `image-organizer`, `golang`, `sqlite`, `wails`, `cli`

GitHub의 `Settings > General > Repository name`에서 이름을 바꾼 뒤 로컬 원격 주소를 갱신한다. GitHub는 기존 저장소 URL을 새 URL로 리디렉션하지만, 로컬 설정은 명시적으로 바꾸는 편이 명확하다.

After renaming the repository under `Settings > General > Repository name`, update the local remote URL. GitHub redirects the previous repository URL, but updating the local configuration avoids ambiguity.

```powershell
git remote set-url origin https://github.com/soira237-risu/imv.git
git remote -v
```

## 자동 배포 / Automated distribution

`.github/workflows/release.yml`은 `vMAJOR.MINOR.PATCH` 형식의 태그가 푸시될 때 Windows x64 릴리스를 만든다.

`.github/workflows/release.yml` creates a Windows x64 release when a `vMAJOR.MINOR.PATCH` tag is pushed.

릴리스 ZIP에는 다음 파일이 들어간다.

The release ZIP contains:

- `imv.exe`: CLI
- `imv-gui.exe`: 선택형 Wails 데스크톱 GUI / optional Wails desktop GUI
- `README.md`

ZIP과 별도로 SHA-256 체크섬 파일도 Release에 첨부된다.

A SHA-256 checksum file is attached to the Release alongside the ZIP.

## 릴리스 절차 / Release procedure

1. 변경을 `main`에 병합하고 CI가 통과했는지 확인한다. / Merge changes into `main` and confirm CI passes.
2. 버전 태그를 만들고 푸시한다. / Create and push a version tag.
3. GitHub Actions의 `Release` 작업이 성공했는지 확인한다. / Confirm the `Release` workflow succeeds.
4. GitHub Release에서 ZIP을 내려받아 새 폴더에서 CLI 도움말과 GUI 실행을 확인한다. / Download the ZIP from the GitHub Release and smoke-test both executables in a clean folder.

```powershell
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "imv v0.1.0"
git push origin v0.1.0
```

실패한 태그를 같은 커밋에서 다시 실행할 때는 GitHub Actions의 `Re-run jobs`를 사용한다. 이미 공개된 버전을 다른 커밋으로 교체하지 말고 새 패치 버전을 만든다.

Use `Re-run jobs` in GitHub Actions to retry a failed workflow for the same commit. Do not move an already published version tag to another commit; publish a new patch version instead.

## Windows 배포 주의점 / Windows distribution notes

- GUI는 Windows 10/11의 Microsoft Edge WebView2 Runtime을 사용한다. 대부분의 최신 Windows에는 이미 설치되어 있다.
- 현재 실행 파일은 코드 서명되지 않는다. Windows SmartScreen 경고가 표시될 수 있으며, 공개 사용자가 늘면 코드 서명 인증서와 서명 단계를 추가한다.
- 최초 배포는 `windows-amd64`만 제공한다. 실제 수요가 확인되면 Windows ARM64와 다른 운영체제를 별도 작업으로 추가한다.

- The GUI uses Microsoft Edge WebView2 Runtime on Windows 10/11; it is already present on most current systems.
- The executables are currently unsigned, so Windows SmartScreen may display a warning. Add a code-signing certificate and signing step when broader public distribution warrants it.
- The initial release targets `windows-amd64`. Add Windows ARM64 or other operating systems as separate jobs when demand is confirmed.
