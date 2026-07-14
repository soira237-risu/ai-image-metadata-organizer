import { Database, Tags } from "lucide-react";
import type { LibraryController } from "./controller";
import { countMap } from "./format";

const sourceOptions = [["", "전체"], ["nai", "NovelAI"], ["comfyui", "ComfyUI"], ["generic", "Generic"], ["unknown", "Unknown"]];
const formatOptions = [["", "전체"], ["png", "PNG"], ["webp", "WebP"]];

export function FilterSidebar({ controller }: { controller: LibraryController }) {
  const { filters, stats, tags, actions } = controller;
  return (
    <aside className="filters" aria-label="라이브러리 필터">
      <section className="filter-section">
        <div className="section-title"><span>필터</span><button type="button" className="text-button" onClick={() => {
          actions.updateFilter("tag", "");
          actions.updateFilter("source", "");
          actions.updateFilter("format", "");
          actions.updateFilter("hasWorkflow", false);
        }}>초기화</button></div>
        <label>태그<input data-testid="tag-input" value={filters.tag} onChange={(event) => actions.selectTag(event.target.value)} placeholder="예: blue hair" /></label>
        <label>생성 도구<select value={filters.source} onChange={(event) => actions.updateFilter("source", event.target.value)}>{sourceOptions.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label>파일 형식<select value={filters.format} onChange={(event) => actions.updateFilter("format", event.target.value)}>{formatOptions.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
        <label className="check-row"><input type="checkbox" checked={filters.hasWorkflow} onChange={(event) => actions.updateFilter("hasWorkflow", event.target.checked)} /><span>워크플로 포함</span></label>
      </section>

      <section className="side-card stats-block">
        <div className="side-card-title"><Database size={15} /><h2>라이브러리</h2></div>
        <div className="stat-pair"><span><strong>{stats?.total_files ?? "—"}</strong><small>파일</small></span><span><strong>{stats?.workflow_files ?? "—"}</strong><small>워크플로</small></span></div>
        <p>{stats ? countMap(stats.formats) : "아직 통계가 없습니다"}</p>
      </section>

      <section className="tags-block">
        <div className="side-card-title"><Tags size={15} /><h2>인기 태그</h2></div>
        <div className="tag-cloud">
          {tags.slice(0, 14).map((tag) => <button className={filters.tag === tag.tag ? "tag-button active" : "tag-button"} type="button" key={tag.tag} onClick={() => actions.selectTag(tag.tag)}><span>{tag.tag}</span><small>{tag.count}</small></button>)}
        </div>
      </section>
    </aside>
  );
}
