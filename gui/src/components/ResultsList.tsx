import { FileImage, LoaderCircle, Sparkles } from "lucide-react";
import type { LibraryController } from "./controller";
import { dimensions, fileName, promptPreview, recordSource, tagPreview } from "./format";

export function ResultsList({ controller }: { controller: LibraryController }) {
  const { records, selectedID, folderState, isSearching, actions } = controller;
  return (
    <section className="results" aria-label="이미지 결과">
      <div className="results-head">
        <div><strong>이미지</strong><span className="count-badge">{records.length}</span>{isSearching && <LoaderCircle className="spin" size={14} />}</div>
        <small title={folderState.db_path}>최신 색인 기준</small>
      </div>
      <div className="record-list" data-testid="record-list">
        {records.map((record) => (
          <button key={record.id} type="button" className={record.id === selectedID ? "record-row selected" : "record-row"} onClick={() => actions.selectRecord(record.id)} aria-label={`${fileName(record.path)} 선택`}>
            <span className="file-icon"><FileImage size={18} /></span>
            <span className="record-main"><strong>{fileName(record.path)}</strong><small>{promptPreview(record) || "프롬프트 메타데이터 없음"}</small><span className="record-tags">{tagPreview(record)}</span></span>
            <span className="record-meta"><em>{recordSource(record)}</em><small>{record.format.toUpperCase()}</small><small>{dimensions(record)}</small>{record.metadata?.some((item) => item.workflow_summary && Object.keys(item.workflow_summary).length > 0) && <Sparkles size={13} />}</span>
          </button>
        ))}
        {records.length === 0 && <div className="empty-state"><FileImage size={28} /><strong>색인된 이미지가 없습니다</strong><span>폴더를 열고 스캔하면 여기에 표시됩니다.</span></div>}
      </div>
    </section>
  );
}
