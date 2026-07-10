import { useCallback, useEffect, useMemo, useState } from "react";
import type { CSSProperties, PointerEvent as ReactPointerEvent } from "react";
import {
  applyMove,
  exportJSON,
  getImage,
  getState,
  getStats,
  getTags,
  inspectFile,
  openFile,
  openFolder,
  planMove,
  previewPath,
  resetSession,
  revealFolder,
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
  TagRecord,
  TagSummary
} from "./types";
import "./styles.css";

const sourceOptions = ["", "nai", "comfyui", "generic", "unknown"];
const formatOptions = ["", "png", "webp"];
const densityOptions = ["컴팩트", "기본", "넓게"] as const;

type Filters = {
  query: string;
  tag: string;
  source: string;
  format: string;
  hasWorkflow: boolean;
};

type OptionsTab = "display" | "layout" | "help";

type DensityOption = typeof densityOptions[number];

type LayoutSettings = {
  fontSize: number;
  gap: number;
  filterWidth: number;
  detailWidth: number;
  previewHeight: number;
  density: DensityOption;
};

const emptyFilters: Filters = {
  query: "",
  tag: "",
  source: "",
  format: "",
  hasWorkflow: false
};

const defaultLayoutSettings: LayoutSettings = {
  fontSize: 14,
  gap: 10,
  filterWidth: 220,
  detailWidth: 420,
  previewHeight: 250,
  density: "기본"
};

const densityRows: Record<DensityOption, number> = {
  "컴팩트": 58,
  "기본": 72,
  "넓게": 88
};

export default function App() {
  const [folderState, setFolderState] = useState<FolderState>({ folder: "", db_path: ".imv/imv.db" });
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [records, setRecords] = useState<ImageRecord[]>([]);
  const [tags, setTags] = useState<TagSummary[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [detail, setDetail] = useState<ImageDetail | null>(null);
  const [singleFilePath, setSingleFilePath] = useState("");
  const [progress, setProgress] = useState<ScanProgress | null>(null);
  const [status, setStatus] = useState("준비됨");
  const [isScanning, setScanning] = useState(false);
  const [isOpenMenuVisible, setOpenMenuVisible] = useState(false);
  const [isOptionsVisible, setOptionsVisible] = useState(false);
  const [optionsTab, setOptionsTab] = useState<OptionsTab>("display");
  const [layoutSettings, setLayoutSettings] = useState<LayoutSettings>(defaultLayoutSettings);
  const [moveTag, setMoveTag] = useState("");
  const [moveTo, setMoveTo] = useState("");
  const [conflict, setConflict] = useState<MoveRequest["conflict"]>("skip");
  const [movePlans, setMovePlans] = useState<MovePlan[]>([]);
  const [moveResult, setMoveResult] = useState<MovePlan[] | null>(null);
  const [movePreviews, setMovePreviews] = useState<Record<string, string>>({});

  const refreshRecords = useCallback(async (nextFilters = filters, preferredPath = "") => {
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
    const preferred = preferredPath ? found.find((item) => samePath(item.path, preferredPath)) : undefined;
    if (preferred) {
      setSelectedID(preferred.id);
      return found;
    }
    if (found.length > 0 && !found.some((item) => item.id === selectedID)) {
      setSelectedID(found[0].id);
    }
    if (found.length === 0) {
      setSelectedID(null);
      setDetail(null);
    }
    return found;
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
    if (singleFilePath) {
      return;
    }
    void refreshRecords().catch((error) => setStatus(messageFrom(error)));
  }, [filters.query, filters.tag, filters.source, filters.format, filters.hasWorkflow, singleFilePath]);

  useEffect(() => {
    if (singleFilePath) {
      return;
    }
    void refreshSideData().catch(() => undefined);
  }, [refreshSideData, singleFilePath]);

  useEffect(() => {
    if (selectedID === null) {
      setDetail(null);
      return;
    }
    if (selectedID <= 0) {
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
    const loadDroppedImage = (paths: string[]) => {
      if (paths.length > 0) {
        void inspectSingleFile(paths[0]);
      }
    };
    if (window.runtime?.OnFileDrop) {
      window.runtime.OnFileDrop((_x, _y, paths) => {
        loadDroppedImage(paths);
      }, false);
    } else {
      window.runtime?.EventsOn?.("wails:file-drop", (...args) => {
        loadDroppedImage(extractDroppedPaths(args));
      });
    }
    return () => {
      window.runtime?.OnFileDropOff?.();
    };
  }, []);

  useEffect(() => {
    if (!moveResult) {
      setMovePreviews({});
      return;
    }
    const plans = moveResult;
    let cancelled = false;
    async function loadPreviews() {
      const entries = await Promise.all(plans.map(async (plan) => {
        const path = plan.destination_path || plan.source_path;
        try {
          return [plan.destination_path, await previewPath(path)] as const;
        } catch {
          return [plan.destination_path, ""] as const;
        }
      }));
      if (!cancelled) {
        setMovePreviews(Object.fromEntries(entries));
      }
    }
    void loadPreviews();
    return () => {
      cancelled = true;
    };
  }, [moveResult]);

  const selected = useMemo(() => {
    return records.find((record) => record.id === selectedID) ?? detail?.record ?? null;
  }, [records, selectedID, detail]);

  const appStyle = useMemo(() => ({
    "--ui-font-size": `${layoutSettings.fontSize}px`,
    "--ui-gap": `${layoutSettings.gap}px`,
    "--filter-width": `${layoutSettings.filterWidth}px`,
    "--detail-width": `${layoutSettings.detailWidth}px`,
    "--preview-height": `${layoutSettings.previewHeight}px`,
    "--record-row-min": `${densityRows[layoutSettings.density]}px`
  }) as CSSProperties, [layoutSettings]);

  function updateLayout<K extends keyof LayoutSettings>(key: K, value: LayoutSettings[K]) {
    setLayoutSettings((current) => ({ ...current, [key]: value }));
  }

  function startPaneResize(kind: "filter" | "detail", event: ReactPointerEvent<HTMLButtonElement>) {
    event.preventDefault();
    const startX = event.clientX;
    const startWidth = kind === "filter" ? layoutSettings.filterWidth : layoutSettings.detailWidth;
    const onMove = (moveEvent: PointerEvent) => {
      const delta = moveEvent.clientX - startX;
      setLayoutSettings((current) => {
        if (kind === "filter") {
          return { ...current, filterWidth: clamp(startWidth + delta, 160, 360) };
        }
        return { ...current, detailWidth: clamp(startWidth - delta, 320, 640) };
      });
    };
    const onUp = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  async function chooseFile() {
    setOpenMenuVisible(false);
    try {
      const state = await openFile();
      setFolderState(state);
      if (state.selected_path) {
        await inspectSingleFile(state.selected_path);
      }
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function inspectSingleFile(path: string) {
    setSingleFilePath(path);
    setFolderState({
      folder: pathDir(path),
      db_path: "단일 파일 모드",
      selected_path: path
    });
    setRecords([]);
    setSelectedID(null);
    setDetail(null);
    setTags([]);
    setStats(null);
    setProgress(fileProgress(path, 0));
    setMovePlans([]);
    setMoveResult(null);
    setMovePreviews({});
    setScanning(true);
    setStatus(`파일 불러오는 중: ${path}`);
    try {
      const image = await inspectFile(path);
      setRecords([image.record]);
      setDetail(image);
      setSelectedID(image.record.id);
      setProgress(fileProgress(path, 1));
      setStatus(`파일 불러옴: ${path}`);
    } catch (error) {
      setProgress(fileProgress(path, 1, 1));
      setStatus(messageFrom(error));
    } finally {
      setScanning(false);
    }
  }

  async function chooseFolder() {
    setOpenMenuVisible(false);
    try {
      const state = await openFolder();
      setSingleFilePath("");
      setFolderState(state);
      if (state.folder) {
        setStatus(`폴더: ${state.folder}`);
      }
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function scan(folder = folderState.folder, rescan = false, preferredPath = "") {
    if (!folder) {
      setStatus("먼저 파일이나 폴더를 열어주세요");
      return;
    }
    setScanning(true);
    setSingleFilePath("");
    setStatus("스캔 중");
    setProgress(null);
    try {
      const result = await scanFolder(folder, rescan);
      const nextState = await getState();
      setFolderState(nextState);
      await refreshRecords(filters, preferredPath);
      await refreshSideData();
      setStatus(`색인 ${result.indexed}개, 건너뜀 ${result.skipped}개, 오류 ${result.errors.length}개`);
    } catch (error) {
      setStatus(messageFrom(error));
    } finally {
      setScanning(false);
    }
  }

  async function resetLoadedState() {
    try {
      const state = await resetSession();
      setFolderState(state);
      setSingleFilePath("");
      setFilters(emptyFilters);
      setRecords([]);
      setTags([]);
      setStats(null);
      setSelectedID(null);
      setDetail(null);
      setProgress(null);
      setMoveTag("");
      setMoveTo("");
      setConflict("skip");
      setMovePlans([]);
      setMoveResult(null);
      setMovePreviews({});
      setStatus("초기화됨");
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  function updateFilter<K extends keyof Filters>(key: K, value: Filters[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  async function createMovePlan() {
    if (!moveTag || !moveTo) {
      setStatus("이동 태그와 대상 폴더를 입력해주세요");
      return;
    }
    try {
      const plans = await planMove({ tag: moveTag, to: moveTo, conflict });
      setMovePlans(plans);
      setMoveResult(null);
      setMovePreviews({});
      setStatus(`이동 계획 ${plans.length}개`);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function applyCurrentPlan() {
    if (!moveTag || !moveTo || movePlans.length === 0) {
      return;
    }
    const ok = window.confirm(`${movePlans.length}개 파일을 이동할까요?`);
    if (!ok) {
      return;
    }
    try {
      const plans = await applyMove({ tag: moveTag, to: moveTo, conflict });
      setMovePlans(plans);
      setMoveResult(plans);
      await refreshRecords();
      await refreshSideData();
      setStatus(`이동됨 ${countStatus(plans, "moved")}개`);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function openMoveFolder() {
    const target = moveRevealTarget(moveResult ?? [], moveTo);
    if (!target) {
      return;
    }
    try {
      await revealFolder(target);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function runExport() {
    try {
      const result = await exportJSON(true);
      if (result.path) {
        setStatus(`${result.count}개 레코드 내보냄`);
      }
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  return (
    <main
      className="app-shell"
      data-testid="app-shell"
      style={appStyle}
      onDragOver={(event) => event.preventDefault()}
      onDrop={(event) => event.preventDefault()}
    >
      <header className="toolbar">
        <div className="brand">
          <strong title="AI Image Metadata Organizer">IMV</strong>
          <span>{shortPath(folderState.folder) || "불러온 폴더 없음"}</span>
        </div>
        <div className="toolbar-actions">
          <div className="open-menu">
            <button type="button" onClick={() => setOpenMenuVisible((open) => !open)}>열기</button>
            {isOpenMenuVisible && (
              <div className="open-menu-popover">
                <button type="button" onClick={() => void chooseFile()}>파일 열기</button>
                <button type="button" onClick={() => void chooseFolder()}>폴더 열기</button>
              </div>
            )}
          </div>
          <button type="button" onClick={() => void scan()} disabled={isScanning || !folderState.folder}>스캔</button>
          <button type="button" onClick={() => void scan(folderState.folder, true)} disabled={isScanning || !folderState.folder}>다시 스캔</button>
          <button type="button" onClick={() => void runExport()}>내보내기</button>
          <button type="button" onClick={() => void resetLoadedState()}>초기화</button>
          <button type="button" onClick={() => setOptionsVisible(true)}>옵션</button>
        </div>
      </header>

      <section className="workspace">
        <aside className="filters">
          <label>
            검색어
            <input
              data-testid="query-input"
              value={filters.query}
              onChange={(event) => updateFilter("query", event.target.value)}
              placeholder="프롬프트 또는 설정"
            />
          </label>
          <label>
            태그
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
            소스
            <select value={filters.source} onChange={(event) => updateFilter("source", event.target.value)}>
              {sourceOptions.map((source) => <option key={source} value={source}>{source || "전체"}</option>)}
            </select>
          </label>
          <label>
            형식
            <select value={filters.format} onChange={(event) => updateFilter("format", event.target.value)}>
              {formatOptions.map((format) => <option key={format} value={format}>{format || "전체"}</option>)}
            </select>
          </label>
          <label className="check-row">
            <input
              type="checkbox"
              checked={filters.hasWorkflow}
              onChange={(event) => updateFilter("hasWorkflow", event.target.checked)}
            />
            워크플로우 있음
          </label>

          <div className="stats-block">
            <h2>통계</h2>
            <p>{stats ? `${stats.total_files}개 파일` : "-"}</p>
            <p>{stats ? `${stats.workflow_files}개 워크플로우` : "-"}</p>
            <small>{stats ? countMap(stats.formats) : ""}</small>
          </div>

          <div className="tags-block">
            <h2>태그</h2>
            {tags.map((tag) => (
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
        <PaneHandle label="왼쪽 패널 폭 조절" onPointerDown={(event) => startPaneResize("filter", event)} />

        <section className="results" aria-label="이미지 결과">
          <div className="results-head">
            <span>{records.length}개 결과</span>
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
            {records.length === 0 && <div className="empty">색인된 이미지가 없습니다</div>}
          </div>
        </section>

        <PaneHandle label="상세 패널 폭 조절" onPointerDown={(event) => startPaneResize("detail", event)} />
        <DetailPanel detail={detail} selected={selected} />
      </section>

      <section className="move-panel" aria-label="이동 계획">
        <label>
          이동 태그
          <input data-testid="move-tag-input" value={moveTag} onChange={(event) => setMoveTag(event.target.value)} />
        </label>
        <label>
          대상 폴더
          <input data-testid="move-to-input" value={moveTo} onChange={(event) => setMoveTo(event.target.value)} />
        </label>
        <label>
          충돌 처리
          <select value={conflict} onChange={(event) => setConflict(event.target.value as MoveRequest["conflict"])}>
            <option value="skip">skip</option>
            <option value="rename">rename</option>
          </select>
        </label>
        <button className="move-plan-button" type="button" onClick={() => void createMovePlan()}>계획</button>
        <button className="move-apply-button" type="button" onClick={() => void applyCurrentPlan()} disabled={movePlans.length === 0}>이동 실행</button>
        <div className="plan-summary" data-testid="move-plan-summary">
          {movePlans.length > 0 ? `${movePlans.length}개 계획됨` : "이동 계획 없음"}
        </div>
      </section>

      {isOptionsVisible && (
        <OptionsPopup
          tab={optionsTab}
          settings={layoutSettings}
          onClose={() => setOptionsVisible(false)}
          onTabChange={setOptionsTab}
          onChange={updateLayout}
          onReset={() => setLayoutSettings(defaultLayoutSettings)}
        />
      )}

      <MoveResultPanel
        plans={moveResult ?? []}
        previews={movePreviews}
        onClose={() => setMoveResult(null)}
        onReveal={() => void openMoveFolder()}
      />

      <footer className="statusbar">
        <span>{status}</span>
        <span>{progress ? `${progress.done}/${progress.total}` : ""}</span>
      </footer>
    </main>
  );
}

function PaneHandle({ label, onPointerDown }: { label: string; onPointerDown: (event: ReactPointerEvent<HTMLButtonElement>) => void }) {
  return (
    <button
      className="pane-handle"
      type="button"
      aria-label={label}
      onPointerDown={onPointerDown}
    />
  );
}

function OptionsPopup({
  tab,
  settings,
  onClose,
  onTabChange,
  onChange,
  onReset
}: {
  tab: OptionsTab;
  settings: LayoutSettings;
  onClose: () => void;
  onTabChange: (tab: OptionsTab) => void;
  onChange: <K extends keyof LayoutSettings>(key: K, value: LayoutSettings[K]) => void;
  onReset: () => void;
}) {
  return (
    <div className="options-backdrop">
      <section className="options-popup" role="dialog" aria-label="옵션">
        <header>
          <div>
            <h2>옵션</h2>
            <p>화면 크기와 읽기 밀도를 작업 방식에 맞춥니다.</p>
          </div>
          <div className="options-header-actions">
            <button type="button" onClick={onReset}>기본값</button>
            <button type="button" onClick={onClose}>닫기</button>
          </div>
        </header>
        <div className="options-body">
          <nav className="options-tabs" aria-label="옵션 탭">
            <button type="button" className={tab === "display" ? "active" : ""} onClick={() => onTabChange("display")}>화면</button>
            <button type="button" className={tab === "layout" ? "active" : ""} onClick={() => onTabChange("layout")}>레이아웃</button>
            <button type="button" className={tab === "help" ? "active" : ""} onClick={() => onTabChange("help")}>도움말</button>
          </nav>
          <div className="options-content">
            {tab === "display" && (
              <>
                <RangeControl
                  label="글자 크기"
                  min={12}
                  max={18}
                  value={settings.fontSize}
                  unit="px"
                  onChange={(value) => onChange("fontSize", value)}
                />
                <RangeControl
                  label="간격"
                  min={6}
                  max={18}
                  value={settings.gap}
                  unit="px"
                  onChange={(value) => onChange("gap", value)}
                />
                <RangeControl
                  label="미리보기 높이"
                  min={180}
                  max={380}
                  value={settings.previewHeight}
                  unit="px"
                  onChange={(value) => onChange("previewHeight", value)}
                />
                <DensityControl value={settings.density} onChange={(value) => onChange("density", value)} />
              </>
            )}
            {tab === "layout" && (
              <>
                <RangeControl
                  label="왼쪽 패널 폭"
                  min={160}
                  max={360}
                  value={settings.filterWidth}
                  unit="px"
                  onChange={(value) => onChange("filterWidth", value)}
                />
                <RangeControl
                  label="상세 패널 폭"
                  min={320}
                  max={640}
                  value={settings.detailWidth}
                  unit="px"
                  onChange={(value) => onChange("detailWidth", value)}
                />
                <div className="options-note">
                  <strong>빠른 조절</strong>
                  <p>목록과 상세 사이의 얇은 손잡이를 드래그해도 패널 폭을 바로 바꿀 수 있습니다.</p>
                </div>
              </>
            )}
            {tab === "help" && <HelpContent />}
          </div>
        </div>
      </section>
    </div>
  );
}

function RangeControl({ label, min, max, value, unit, onChange }: { label: string; min: number; max: number; value: number; unit: string; onChange: (value: number) => void }) {
  return (
    <label className="range-control">
      <span>
        {label}
        <strong>{value}{unit}</strong>
      </span>
      <input
        aria-label={label}
        type="range"
        min={min}
        max={max}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      />
    </label>
  );
}

function DensityControl({ value, onChange }: { value: DensityOption; onChange: (value: DensityOption) => void }) {
  return (
    <div className="density-control">
      <span>밀도</span>
      <div className="segmented" role="group" aria-label="밀도">
        {densityOptions.map((option) => (
          <button
            key={option}
            className={value === option ? "active" : ""}
            type="button"
            onClick={() => onChange(option)}
          >
            {option}
          </button>
        ))}
      </div>
    </div>
  );
}

function HelpContent() {
  return (
    <div className="help-grid">
      <article>
        <h3>파일 열기</h3>
        <p>단일 PNG/WebP 파일만 읽어서 바로 보여줍니다. 폴더 전체 스캔은 하지 않습니다.</p>
      </article>
      <article>
        <h3>폴더 열기</h3>
        <p>작업할 폴더를 고른 뒤 스캔하거나 다시 스캔해서 DB 검색 목록을 갱신합니다.</p>
      </article>
      <article>
        <h3>드래그 앤 드롭</h3>
        <p>이미지 파일 하나를 창에 놓으면 파일 열기처럼 단일 파일 모드로 불러옵니다.</p>
      </article>
      <article>
        <h3>이동 실행</h3>
        <p>먼저 이동 태그와 대상 폴더로 계획을 만든 뒤 실행합니다. 결과 패널에서 이동된 파일 미리보기와 대상 폴더 열기를 확인할 수 있습니다.</p>
      </article>
      <article>
        <h3>초기화</h3>
        <p>현재 화면 상태만 비웁니다. DB와 이미지 파일은 삭제하지 않습니다.</p>
      </article>
    </div>
  );
}

function DetailPanel({ detail, selected }: { detail: ImageDetail | null; selected: ImageRecord | null }) {
  const record = detail?.record ?? selected;
  if (!record) {
    return (
      <aside className="detail">
        <div className="empty">선택 없음</div>
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
          <div className="empty">미리보기 없음</div>
        )}
      </div>
      <div className="detail-scroll">
        <div className="detail-head">
          <h1>{fileName(record.path)}</h1>
          <small>{record.path}</small>
        </div>
        <div className="detail-grid">
          <span>형식</span><strong>{record.format}</strong>
          <span>크기</span><strong>{dimensions(record)}</strong>
          <span>파일 용량</span><strong>{formatBytes(record.size)}</strong>
          <span>소스</span><strong>{recordSource(record)}</strong>
        </div>
        <section className="detail-section">
          <h2>태그</h2>
          <TagChips tags={record.tags ?? []} />
        </section>
        {metadata.map((meta) => (
          <section className="detail-section" key={meta.source}>
            <h2>{meta.source}</h2>
            {meta.positive_prompt && <PromptBlock label="긍정 프롬프트" text={meta.positive_prompt} />}
            {meta.negative_prompt && <PromptBlock label="부정 프롬프트" text={meta.negative_prompt} />}
            <KeyValue title="설정" value={meta.settings} />
            <KeyValue title="워크플로우" value={meta.workflow_summary} />
          </section>
        ))}
      </div>
    </aside>
  );
}

function MoveResultPanel({ plans, previews, onClose, onReveal }: { plans: MovePlan[]; previews: Record<string, string>; onClose: () => void; onReveal: () => void }) {
  if (plans.length === 0) {
    return (
      <section className="move-result-panel empty-panel" aria-label="이동 결과">
        <span>이동 결과 없음</span>
      </section>
    );
  }
  return (
      <section className="move-result-panel" role="region" aria-label="이동 결과">
        <header>
          <h2>이동 결과</h2>
          <button type="button" onClick={onClose}>닫기</button>
        </header>
        <div className="move-result-summary">
          <strong>전체 {plans.length}</strong>
          <span>이동됨 {countStatus(plans, "moved")}</span>
          <span>충돌로 건너뜀 {countStatus(plans, "skipped_conflict")}</span>
          <span>실패 {plans.filter((plan) => plan.status.startsWith("failed")).length}</span>
        </div>
        <div className="move-result-list">
          {plans.map((plan) => (
            <div className="move-result-row" key={`${plan.file_id}-${plan.destination_path}`}>
              <div className="move-result-preview">
                {previews[plan.destination_path] ? (
                  <img src={previews[plan.destination_path]} alt={fileName(plan.destination_path)} />
                ) : (
                  <span>미리보기 없음</span>
                )}
              </div>
              <strong>{statusLabel(plan.status)}</strong>
              <span>{plan.source_path}</span>
              <span>{plan.destination_path}</span>
            </div>
          ))}
        </div>
        <footer>
          <button type="button" onClick={onReveal}>이동 폴더 열기</button>
        </footer>
      </section>
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

function TagChips({ tags }: { tags: TagRecord[] }) {
  const values = tags
    .map((tag) => tag.tag || tag.normalized_tag)
    .filter(Boolean);
  if (values.length === 0) {
    return <p>-</p>;
  }
  return (
    <div className="tag-chip-list">
      {values.map((value) => <span className="tag-chip" key={value}>{value}</span>)}
    </div>
  );
}

function extractDroppedPaths(args: unknown[]): string[] {
  const paths: string[] = [];
  for (const arg of args) {
    paths.push(...collectDroppedPaths(arg));
  }
  return paths;
}

function collectDroppedPaths(value: unknown): string[] {
  if (typeof value === "string") {
    return [value];
  }
  if (Array.isArray(value)) {
    return value.flatMap(collectDroppedPaths);
  }
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return ["path", "Path", "paths", "Paths", "file", "File"].flatMap((key) => collectDroppedPaths(record[key]));
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

function pathDir(path: string): string {
  const index = Math.max(path.lastIndexOf("\\"), path.lastIndexOf("/"));
  return index > 0 ? path.slice(0, index) : path;
}

function fileProgress(path: string, done: number, errors = 0): ScanProgress {
  return {
    path,
    total: 1,
    done,
    scanned: done,
    indexed: errors > 0 ? 0 : done,
    skipped: 0,
    errors
  };
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

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, Math.round(value)));
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

function countStatus(plans: MovePlan[], status: string): number {
  return plans.filter((plan) => plan.status === status).length;
}

function statusLabel(status: string): string {
  switch (status) {
    case "moved":
      return "이동됨";
    case "skipped_conflict":
      return "충돌로 건너뜀";
    case "planned":
      return "계획됨";
    default:
      return status.startsWith("failed") ? "실패" : status;
  }
}

function moveRevealTarget(plans: MovePlan[], fallback: string): string {
  const moved = plans.find((plan) => plan.status === "moved");
  if (moved) {
    return pathDir(moved.destination_path);
  }
  return fallback;
}

function samePath(a: string, b: string): boolean {
  return a.replace(/\\/g, "/").toLowerCase() === b.replace(/\\/g, "/").toLowerCase();
}

function messageFrom(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
