import { useCallback, useEffect, useMemo, useState } from "react";
import {
  applyMove,
  exportJSON,
  getImage,
  getState,
  getStats,
  getTags,
  openFolder,
  planMove,
  scanFolder,
  search
} from "./api";
import type {
  FolderState,
  ImageDetail,
  ImageRecord,
  MovePlan,
  MoveRequest,
  ScanProgress,
  SearchRequest,
  Stats,
  TagSummary
} from "./types";
import "./styles.css";

const sourceOptions = ["", "nai", "comfyui", "generic", "unknown"];
const formatOptions = ["", "png", "webp"];

type Filters = {
  query: string;
  tag: string;
  source: string;
  format: string;
  hasWorkflow: boolean;
};

const emptyFilters: Filters = {
  query: "",
  tag: "",
  source: "",
  format: "",
  hasWorkflow: false
};

export default function App() {
  const [folderState, setFolderState] = useState<FolderState>({ folder: "", db_path: ".imv/imv.db" });
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [records, setRecords] = useState<ImageRecord[]>([]);
  const [tags, setTags] = useState<TagSummary[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [detail, setDetail] = useState<ImageDetail | null>(null);
  const [progress, setProgress] = useState<ScanProgress | null>(null);
  const [status, setStatus] = useState("Ready");
  const [isScanning, setScanning] = useState(false);
  const [moveTag, setMoveTag] = useState("");
  const [moveTo, setMoveTo] = useState("");
  const [conflict, setConflict] = useState<MoveRequest["conflict"]>("skip");
  const [movePlans, setMovePlans] = useState<MovePlan[]>([]);

  const refreshRecords = useCallback(async (nextFilters = filters) => {
    const req: SearchRequest = {
      query: nextFilters.query,
      tag: nextFilters.tag,
      source: nextFilters.source,
      format: nextFilters.format,
      has_workflow: nextFilters.hasWorkflow,
      limit: 200
    };
    const found = await search(req);
    setRecords(found);
    if (found.length > 0 && !found.some((item) => item.id === selectedID)) {
      setSelectedID(found[0].id);
    }
    if (found.length === 0) {
      setSelectedID(null);
      setDetail(null);
    }
  }, [filters, selectedID]);

  const refreshSideData = useCallback(async () => {
    const [nextTags, nextStats] = await Promise.all([
      getTags({ source: filters.source, query: filters.tag, limit: 50 }),
      getStats()
    ]);
    setTags(nextTags);
    setStats(nextStats);
  }, [filters.source, filters.tag]);

  useEffect(() => {
    getState()
      .then(setFolderState)
      .catch((error) => setStatus(messageFrom(error)));
  }, []);

  useEffect(() => {
    void refreshRecords().catch((error) => setStatus(messageFrom(error)));
  }, [filters.query, filters.tag, filters.source, filters.format, filters.hasWorkflow]);

  useEffect(() => {
    void refreshSideData().catch(() => undefined);
  }, [refreshSideData]);

  useEffect(() => {
    if (selectedID === null) {
      setDetail(null);
      return;
    }
    getImage(selectedID)
      .then(setDetail)
      .catch((error) => setStatus(messageFrom(error)));
  }, [selectedID]);

  useEffect(() => {
    window.runtime?.EventsOn?.("scan:progress", (item) => {
      setProgress(item as ScanProgress);
    });
    window.runtime?.EventsOn?.("wails:file-drop", (...args) => {
      const paths = extractDroppedPaths(args);
      if (paths.length > 0) {
        void scan(paths[0], false);
      }
    });
  }, []);

  const selected = useMemo(() => {
    return records.find((record) => record.id === selectedID) ?? detail?.record ?? null;
  }, [records, selectedID, detail]);

  async function chooseFolder() {
    try {
      const state = await openFolder();
      setFolderState(state);
      if (state.folder) {
        setStatus(`Folder: ${state.folder}`);
      }
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function scan(folder = folderState.folder, rescan = false) {
    if (!folder) {
      setStatus("Select a folder first");
      return;
    }
    setScanning(true);
    setStatus("Scanning");
    setProgress(null);
    try {
      const result = await scanFolder(folder, rescan);
      const nextState = await getState();
      setFolderState(nextState);
      await refreshRecords();
      await refreshSideData();
      setStatus(`Indexed ${result.indexed}, skipped ${result.skipped}, errors ${result.errors.length}`);
    } catch (error) {
      setStatus(messageFrom(error));
    } finally {
      setScanning(false);
    }
  }

  function updateFilter<K extends keyof Filters>(key: K, value: Filters[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  async function createMovePlan() {
    if (!moveTag || !moveTo) {
      setStatus("Move tag and destination are required");
      return;
    }
    try {
      const plans = await planMove({ tag: moveTag, to: moveTo, conflict });
      setMovePlans(plans);
      setStatus(`Planned ${plans.length} moves`);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function applyCurrentPlan() {
    if (!moveTag || !moveTo || movePlans.length === 0) {
      return;
    }
    const ok = window.confirm(`Move ${movePlans.length} files?`);
    if (!ok) {
      return;
    }
    try {
      const plans = await applyMove({ tag: moveTag, to: moveTo, conflict });
      setMovePlans(plans);
      await refreshRecords();
      await refreshSideData();
      setStatus(`Moved ${plans.filter((item) => item.status === "moved").length} files`);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function runExport() {
    try {
      const result = await exportJSON(true);
      if (result.path) {
        setStatus(`Exported ${result.count} records`);
      }
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  return (
    <main
      className="app-shell"
      onDragOver={(event) => event.preventDefault()}
      onDrop={(event) => event.preventDefault()}
    >
      <header className="toolbar">
        <div className="brand">
          <strong>imv</strong>
          <span>{shortPath(folderState.folder) || "No folder"}</span>
        </div>
        <div className="toolbar-actions">
          <button type="button" onClick={chooseFolder}>Open</button>
          <button type="button" onClick={() => void scan()} disabled={isScanning || !folderState.folder}>Scan</button>
          <button type="button" onClick={() => void scan(folderState.folder, true)} disabled={isScanning || !folderState.folder}>Rescan</button>
          <button type="button" onClick={() => void runExport()}>Export</button>
        </div>
      </header>

      <section className="workspace">
        <aside className="filters">
          <label>
            Query
            <input
              data-testid="query-input"
              value={filters.query}
              onChange={(event) => updateFilter("query", event.target.value)}
              placeholder="prompt or setting"
            />
          </label>
          <label>
            Tag
            <input
              data-testid="tag-input"
              value={filters.tag}
              onChange={(event) => {
                updateFilter("tag", event.target.value);
                setMoveTag(event.target.value);
              }}
              placeholder="blue hair"
            />
          </label>
          <label>
            Source
            <select value={filters.source} onChange={(event) => updateFilter("source", event.target.value)}>
              {sourceOptions.map((source) => <option key={source} value={source}>{source || "any"}</option>)}
            </select>
          </label>
          <label>
            Format
            <select value={filters.format} onChange={(event) => updateFilter("format", event.target.value)}>
              {formatOptions.map((format) => <option key={format} value={format}>{format || "any"}</option>)}
            </select>
          </label>
          <label className="check-row">
            <input
              type="checkbox"
              checked={filters.hasWorkflow}
              onChange={(event) => updateFilter("hasWorkflow", event.target.checked)}
            />
            Workflow
          </label>

          <div className="stats-block">
            <h2>Stats</h2>
            <p>{stats ? `${stats.total_files} files` : "-"}</p>
            <p>{stats ? `${stats.workflow_files} workflows` : "-"}</p>
            <small>{stats ? countMap(stats.formats) : ""}</small>
          </div>

          <div className="tags-block">
            <h2>Tags</h2>
            {tags.slice(0, 12).map((tag) => (
              <button
                className="tag-button"
                type="button"
                key={tag.tag}
                onClick={() => {
                  updateFilter("tag", tag.tag);
                  setMoveTag(tag.tag);
                }}
              >
                <span>{tag.tag}</span>
                <small>{tag.count}</small>
              </button>
            ))}
          </div>
        </aside>

        <section className="results" aria-label="Image results">
          <div className="results-head">
            <span>{records.length} results</span>
            <small>{folderState.db_path}</small>
          </div>
          <div className="record-list" data-testid="record-list">
            {records.map((record) => (
              <button
                key={record.id}
                type="button"
                className={record.id === selectedID ? "record-row selected" : "record-row"}
                onClick={() => setSelectedID(record.id)}
              >
                <span className="record-main">
                  <strong>{fileName(record.path)}</strong>
                  <small>{promptPreview(record)}</small>
                </span>
                <span className="record-meta">
                  <small>{recordSource(record)}</small>
                  <small>{record.format}</small>
                  <small>{dimensions(record)}</small>
                </span>
                <span className="record-tags">{tagPreview(record)}</span>
              </button>
            ))}
            {records.length === 0 && <div className="empty">No indexed images</div>}
          </div>
        </section>

        <DetailPanel detail={detail} selected={selected} />
      </section>

      <section className="move-panel" aria-label="Move plan">
        <label>
          Move tag
          <input data-testid="move-tag-input" value={moveTag} onChange={(event) => setMoveTag(event.target.value)} />
        </label>
        <label>
          To
          <input data-testid="move-to-input" value={moveTo} onChange={(event) => setMoveTo(event.target.value)} />
        </label>
        <label>
          Conflict
          <select value={conflict} onChange={(event) => setConflict(event.target.value as MoveRequest["conflict"])}>
            <option value="skip">skip</option>
            <option value="rename">rename</option>
          </select>
        </label>
        <button type="button" onClick={() => void createMovePlan()}>Plan</button>
        <button type="button" onClick={() => void applyCurrentPlan()} disabled={movePlans.length === 0}>Apply</button>
        <div className="plan-summary" data-testid="move-plan-summary">
          {movePlans.length > 0 ? `${movePlans.length} planned` : "No move plan"}
        </div>
      </section>

      <footer className="statusbar">
        <span>{status}</span>
        <span>{progress ? `${progress.done}/${progress.total}` : ""}</span>
      </footer>
    </main>
  );
}

function DetailPanel({ detail, selected }: { detail: ImageDetail | null; selected: ImageRecord | null }) {
  const record = detail?.record ?? selected;
  if (!record) {
    return (
      <aside className="detail">
        <div className="empty">No selection</div>
      </aside>
    );
  }
  const metadata = record.metadata ?? [];
  return (
    <aside className="detail" data-testid="detail-panel">
      <div className="preview-frame">
        {detail?.preview_data_url ? (
          <img src={detail.preview_data_url} alt={fileName(record.path)} />
        ) : (
          <div className="empty">Preview unavailable</div>
        )}
      </div>
      <div className="detail-head">
        <h1>{fileName(record.path)}</h1>
        <small>{record.path}</small>
      </div>
      <div className="detail-grid">
        <span>Format</span><strong>{record.format}</strong>
        <span>Dimensions</span><strong>{dimensions(record)}</strong>
        <span>Size</span><strong>{formatBytes(record.size)}</strong>
        <span>Source</span><strong>{recordSource(record)}</strong>
      </div>
      <section className="detail-section">
        <h2>Tags</h2>
        <p>{tagPreview(record, 24) || "-"}</p>
      </section>
      {metadata.map((meta) => (
        <section className="detail-section" key={meta.source}>
          <h2>{meta.source}</h2>
          {meta.positive_prompt && <PromptBlock label="Positive" text={meta.positive_prompt} />}
          {meta.negative_prompt && <PromptBlock label="Negative" text={meta.negative_prompt} />}
          <KeyValue title="Settings" value={meta.settings} />
          <KeyValue title="Workflow" value={meta.workflow_summary} />
        </section>
      ))}
    </aside>
  );
}

function PromptBlock({ label, text }: { label: string; text: string }) {
  return (
    <div className="prompt-block">
      <span>{label}</span>
      <p>{text}</p>
    </div>
  );
}

function KeyValue({ title, value }: { title: string; value?: Record<string, unknown> }) {
  if (!value || Object.keys(value).length === 0) {
    return null;
  }
  return (
    <div className="kv-block">
      <span>{title}</span>
      <pre>{JSON.stringify(value, null, 2)}</pre>
    </div>
  );
}

function extractDroppedPaths(args: unknown[]): string[] {
  for (const arg of args) {
    if (Array.isArray(arg) && arg.every((item) => typeof item === "string")) {
      return arg as string[];
    }
  }
  return [];
}

function recordSource(record: ImageRecord): string {
  return record.metadata?.[0]?.source || "-";
}

function dimensions(record: ImageRecord): string {
  return record.width > 0 && record.height > 0 ? `${record.width}x${record.height}` : "-";
}

function promptPreview(record: ImageRecord): string {
  const prompt = record.metadata?.find((meta) => meta.positive_prompt)?.positive_prompt ?? "";
  return truncate(prompt.replace(/\s+/g, " "), 120);
}

function tagPreview(record: ImageRecord, max = 8): string {
  return (record.tags ?? [])
    .slice(0, max)
    .map((tag) => tag.tag || tag.normalized_tag)
    .filter(Boolean)
    .join(", ");
}

function fileName(path: string): string {
  return path.split(/[\\/]/).pop() || path;
}

function shortPath(path: string): string {
  if (path.length <= 70) {
    return path;
  }
  return `...${path.slice(path.length - 67)}`;
}

function truncate(value: string, max: number): string {
  if (value.length <= max) {
    return value;
  }
  return `${value.slice(0, max - 3)}...`;
}

function countMap(value: Record<string, number>): string {
  return Object.entries(value)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, count]) => `${key} ${count}`)
    .join(" / ");
}

function formatBytes(value: number): string {
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function messageFrom(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
