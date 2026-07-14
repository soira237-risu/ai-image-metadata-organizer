import { Download, FolderOpen, Image, RefreshCw, ScanSearch, Search, X } from "lucide-react";
import type { LibraryController } from "./controller";
import { shortPath } from "./format";

export function Toolbar({ controller }: { controller: LibraryController }) {
  const { folderState, filters, isScanning, isApplyingMove, actions } = controller;
  return (
    <header className="toolbar">
      <div className="brand" aria-label="imv 이미지 메타데이터 뷰어">
        <span className="brand-mark"><Image size={20} strokeWidth={2.2} /></span>
        <span className="brand-copy"><strong>imv</strong><small>{shortPath(folderState.folder) || "로컬 이미지 라이브러리"}</small></span>
      </div>
      <label className="global-search">
        <Search size={17} aria-hidden="true" />
        <span className="sr-only">프롬프트 및 설정 검색</span>
        <input
          data-testid="query-input"
          value={filters.query}
          onChange={(event) => actions.updateFilter("query", event.target.value)}
          placeholder="프롬프트, 설정, 모델 검색"
        />
      </label>
      <div className="toolbar-actions">
        <button className="button secondary" data-testid="open-folder-button" type="button" disabled={isScanning || isApplyingMove} onClick={() => void actions.chooseFolder()} title="이미지 폴더 열기">
          <FolderOpen size={16} /> <span>폴더 열기</span>
        </button>
        {isScanning ? (
          <button className="button danger-soft" data-testid="cancel-scan-button" type="button" onClick={() => void actions.cancelScan()}>
            <X size={16} /> <span>스캔 취소</span>
          </button>
        ) : (
          <button className="button primary" data-testid="scan-button" type="button" disabled={!folderState.folder || isApplyingMove} onClick={() => void actions.scan()}>
            <ScanSearch size={16} /> <span>스캔</span>
          </button>
        )}
        <button className="icon-button" type="button" disabled={isScanning || !folderState.folder || isApplyingMove} onClick={() => void actions.scan(folderState.folder, true)} title="전체 다시 스캔" aria-label="전체 다시 스캔">
          <RefreshCw size={17} />
        </button>
        <button className="icon-button" type="button" disabled={isScanning || isApplyingMove} onClick={() => void actions.runExport()} title="JSON 내보내기" aria-label="JSON 내보내기">
          <Download size={17} />
        </button>
      </div>
    </header>
  );
}
