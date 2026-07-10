# 보안 안내 / Security

데이터 손실, 로컬 경로 노출, 파일 이동 문제, 또는 기타 보안 취약점을 발견했다면 GitHub의 [Private vulnerability reporting](https://github.com/soira237-risu/ai-image-metadata-organizer/security/advisories/new)을 사용해 비공개로 신고해 주세요.

For data-loss, path-exposure, unsafe file-move, or other security reports, use [GitHub Private vulnerability reporting](https://github.com/soira237-risu/ai-image-metadata-organizer/security/advisories/new).

공개 이슈에는 개인 이미지, prompt, SQLite DB, 로컬 경로를 첨부하지 마세요. 재현에 필요한 최소 정보만 비공개 신고에 포함하고, 민감한 원본 파일 대신 안전하게 만든 설명과 재현 절차를 제공하세요.

Do not attach private images, prompts, databases, or local paths to public issues. Include only the minimum reproduction details in the private report, using sanitized descriptions and steps instead of sensitive source files.

긴급한 데이터 손실이 진행 중이면 먼저 파일 작업을 중단하고, CLI `move`의 `--apply` 또는 GUI의 `이동 실행`을 추가로 실행하지 마세요.

If data loss may be ongoing, stop file operations first and do not run CLI `move --apply` or click GUI `이동 실행` again.
