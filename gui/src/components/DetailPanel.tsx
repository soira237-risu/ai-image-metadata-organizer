import { Copy, ImageOff } from "lucide-react";
import type { ImageDetail, ImageRecord } from "../types";
import { dimensions, fileName, formatBytes, recordSource, tagPreview } from "./format";

export function DetailPanel({ detail, selected }: { detail: ImageDetail | null; selected: ImageRecord | null }) {
  const record = detail?.record ?? selected;
  if (!record) return <aside className="detail"><div className="empty-state"><ImageOff size={30} /><strong>이미지를 선택하세요</strong><span>메타데이터와 미리보기가 여기에 표시됩니다.</span></div></aside>;
  const metadata = record.metadata ?? [];
  return (
    <aside className="detail" data-testid="detail-panel">
      <div className="preview-frame">{detail?.preview_data_url ? <img src={detail.preview_data_url} alt={fileName(record.path)} /> : <div className="preview-placeholder"><ImageOff size={28} /><span>미리보기 없음</span></div>}</div>
      <div className="detail-head"><div><h1>{fileName(record.path)}</h1><small title={record.path}>{record.path}</small></div><button className="icon-button" type="button" title="경로 복사" aria-label="경로 복사" onClick={() => void navigator.clipboard?.writeText(record.path)}><Copy size={15} /></button></div>
      <div className="detail-grid"><span>형식</span><strong>{record.format.toUpperCase()}</strong><span>크기</span><strong>{dimensions(record)}</strong><span>파일</span><strong>{formatBytes(record.size)}</strong><span>소스</span><strong>{recordSource(record)}</strong></div>
      <section className="detail-section"><h2>태그</h2><p className="tag-copy">{tagPreview(record, 24) || "—"}</p></section>
      {metadata.map((meta) => <section className="detail-section" key={meta.source}><div className="metadata-title"><h2>{meta.source === "nai" ? "NovelAI" : meta.source === "comfyui" ? "ComfyUI" : meta.source}</h2></div>{meta.positive_prompt && <PromptBlock label="긍정 프롬프트" text={meta.positive_prompt} />}{meta.negative_prompt && <PromptBlock label="부정 프롬프트" text={meta.negative_prompt} />}<KeyValue title="생성 설정" value={meta.settings} /><KeyValue title="워크플로 요약" value={meta.workflow_summary} /></section>)}
    </aside>
  );
}

function PromptBlock({ label, text }: { label: string; text: string }) {
  return <div className="prompt-block"><span>{label}</span><p>{text}</p></div>;
}

function KeyValue({ title, value }: { title: string; value?: Record<string, unknown> }) {
  if (!value || Object.keys(value).length === 0) return null;
  return <details className="kv-block"><summary>{title}</summary><pre>{JSON.stringify(value, null, 2)}</pre></details>;
}
