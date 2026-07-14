import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  applyMove,
  cancelScan as cancelBackendScan,
  chooseDestinationFolder,
  exportJSON,
  getImage,
  getState,
  getStats,
  getTags,
  openFolder,
  planMove,
  scanFolder,
  search
} from "../api";
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
} from "../types";

export type Filters = {
  query: string;
  tag: string;
  source: string;
  format: string;
  hasWorkflow: boolean;
};

export type MoveDraft = Required<MoveRequest>;

type ReviewedMove = {
  request: MoveDraft;
  plans: MovePlan[];
};

const emptyFilters: Filters = {
  query: "",
  tag: "",
  source: "",
  format: "",
  hasWorkflow: false
};

const emptyFolder: FolderState = { folder: "", db_path: ".imv/imv.db" };

export function useLibraryController() {
  const [folderState, setFolderState] = useState<FolderState>(emptyFolder);
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const debouncedFilters = useDebouncedValue(filters, 200);
  const [records, setRecords] = useState<ImageRecord[]>([]);
  const [tags, setTags] = useState<TagSummary[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [detail, setDetail] = useState<ImageDetail | null>(null);
  const [progress, setProgress] = useState<ScanProgress | null>(null);
  const [status, setStatus] = useState("준비됨");
  const [isScanning, setScanning] = useState(false);
  const [isSearching, setSearching] = useState(false);
  const [libraryVersion, setLibraryVersion] = useState(0);
  const [moveDraft, setMoveDraft] = useState<MoveDraft>({ tag: "", to: "", conflict: "skip" });
  const [reviewedMove, setReviewedMove] = useState<ReviewedMove | null>(null);
  const [movePlanInvalidated, setMovePlanInvalidated] = useState(false);
  const [moveDialogOpen, setMoveDialogOpen] = useState(false);
  const [isPlanningMove, setPlanningMove] = useState(false);
  const [isApplyingMove, setApplyingMove] = useState(false);
  const searchSequence = useRef(0);
  const detailSequence = useRef(0);
  const sideSequence = useRef(0);
  const scanSequence = useRef(0);
  const movePlanSequence = useRef(0);
  const movePlanAttempted = useRef(false);

  const clearLibrary = useCallback(() => {
    searchSequence.current += 1;
    detailSequence.current += 1;
    sideSequence.current += 1;
    movePlanSequence.current += 1;
    movePlanAttempted.current = false;
    setRecords([]);
    setTags([]);
    setStats(null);
    setSelectedID(null);
    setDetail(null);
    setReviewedMove(null);
    setMovePlanInvalidated(false);
    setPlanningMove(false);
    setMoveDialogOpen(false);
  }, []);

  useEffect(() => {
    let active = true;
    void getState()
      .then((state) => {
        if (active) setFolderState(state);
      })
      .catch((error) => {
        if (active) setStatus(messageFrom(error));
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const sequence = ++searchSequence.current;
    const request = toSearchRequest(debouncedFilters);
    setSearching(true);
    void search(request)
      .then((found) => {
        if (sequence !== searchSequence.current) return;
        setRecords(found);
        setSelectedID((current) => {
          if (found.length === 0) return null;
          return found.some((item) => item.id === current) ? current : found[0].id;
        });
        if (found.length === 0) setDetail(null);
      })
      .catch((error) => {
        if (sequence === searchSequence.current) setStatus(messageFrom(error));
      })
      .finally(() => {
        if (sequence === searchSequence.current) setSearching(false);
      });
  }, [debouncedFilters, libraryVersion]);

  useEffect(() => {
    const sequence = ++sideSequence.current;
    void Promise.all([
      getTags({ source: debouncedFilters.source, query: debouncedFilters.tag, limit: 50 }),
      getStats()
    ])
      .then(([nextTags, nextStats]) => {
        if (sequence !== sideSequence.current) return;
        setTags(nextTags);
        setStats(nextStats);
      })
      .catch((error) => {
        if (sequence === sideSequence.current) setStatus(messageFrom(error));
      });
  }, [debouncedFilters.source, debouncedFilters.tag, libraryVersion]);

  useEffect(() => {
    const sequence = ++detailSequence.current;
    setDetail(null);
    if (selectedID === null) return;
    void getImage(selectedID)
      .then((nextDetail) => {
        if (sequence === detailSequence.current) setDetail(nextDetail);
      })
      .catch((error) => {
        if (sequence === detailSequence.current) setStatus(messageFrom(error));
      });
  }, [selectedID]);

  const scan = useCallback(async (folder = folderState.folder, rescan = false) => {
    if (!folder) {
      setStatus("먼저 이미지 폴더를 선택하세요");
      return;
    }
    const sequence = ++scanSequence.current;
    setScanning(true);
    setProgress(null);
    setStatus(rescan ? "전체 다시 스캔 중" : "새 파일을 스캔 중");
    try {
      const result = await scanFolder(folder, rescan);
      if (sequence !== scanSequence.current) return;
      setFolderState(await getState());
      setLibraryVersion((value) => value + 1);
      setStatus(`색인 ${result.indexed} · 건너뜀 ${result.skipped} · 오류 ${result.errors.length}`);
    } catch (error) {
      if (sequence !== scanSequence.current) return;
      const message = messageFrom(error);
      setStatus(message.toLowerCase().includes("cancel") ? "스캔이 취소되었습니다" : message);
    } finally {
      if (sequence === scanSequence.current) setScanning(false);
    }
  }, [folderState.folder]);

  useWailsEvents(setProgress, (path) => {
    void scan(path, false);
  });

  const selected = useMemo(() => {
    return records.find((record) => record.id === selectedID) ?? detail?.record ?? null;
  }, [detail, records, selectedID]);

  async function chooseFolder() {
    try {
      const state = await openFolder();
      if (!state.folder) return;
      clearLibrary();
      setFolderState(state);
      setStatus(`폴더를 열었습니다 · ${state.folder}`);
      setLibraryVersion((value) => value + 1);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function cancelScan() {
    try {
      if (await cancelBackendScan()) setStatus("스캔 취소 요청을 보냈습니다");
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  function updateFilter<K extends keyof Filters>(key: K, value: Filters[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  function updateMoveDraft<K extends keyof MoveDraft>(key: K, value: MoveDraft[K]) {
    movePlanSequence.current += 1;
    setMoveDraft((current) => ({ ...current, [key]: value }));
    if (movePlanAttempted.current) {
      setReviewedMove(null);
      setMovePlanInvalidated(true);
      setPlanningMove(false);
      setMoveDialogOpen(false);
    }
  }

  async function browseMoveDestination() {
    try {
      const destination = await chooseDestinationFolder();
      if (destination) updateMoveDraft("to", destination);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  async function createMovePlan() {
    if (!moveDraft.tag.trim() || !moveDraft.to.trim()) {
      setStatus("정리할 태그와 대상 폴더를 입력하세요");
      return;
    }
    const request = { ...moveDraft, tag: moveDraft.tag.trim(), to: moveDraft.to.trim() };
    const sequence = ++movePlanSequence.current;
    movePlanAttempted.current = true;
    setPlanningMove(true);
    setMovePlanInvalidated(false);
    try {
      const plans = await planMove(request);
      if (sequence !== movePlanSequence.current) return;
      setReviewedMove({ request, plans });
      setMovePlanInvalidated(false);
      setStatus(`${plans.length}개 파일의 이동 계획을 만들었습니다`);
    } catch (error) {
      if (sequence === movePlanSequence.current) setStatus(messageFrom(error));
    } finally {
      if (sequence === movePlanSequence.current) setPlanningMove(false);
    }
  }

  async function confirmMove() {
    if (!reviewedMove) return;
    const request = reviewedMove.request;
    setMoveDialogOpen(false);
    setApplyingMove(true);
    try {
      const plans = await applyMove(request);
      setReviewedMove(null);
      movePlanAttempted.current = false;
      setMovePlanInvalidated(false);
      setLibraryVersion((value) => value + 1);
      setStatus(`${plans.filter((item) => item.status === "moved").length}개 파일을 이동했습니다`);
    } catch (error) {
      setStatus(messageFrom(error));
    } finally {
      setApplyingMove(false);
    }
  }

  async function runExport() {
    try {
      const result = await exportJSON(true);
      if (result.path) setStatus(`${result.count}개 레코드를 내보냈습니다 · ${result.path}`);
    } catch (error) {
      setStatus(messageFrom(error));
    }
  }

  function selectTag(tag: string) {
    updateFilter("tag", tag);
    updateMoveDraft("tag", tag);
  }

  return {
    folderState,
    filters,
    records,
    tags,
    stats,
    selectedID,
    selected,
    detail,
    progress,
    status,
    isScanning,
    isSearching,
    moveDraft,
    reviewedMove,
    movePlanInvalidated,
    moveDialogOpen,
    isPlanningMove,
    isApplyingMove,
    actions: {
      chooseFolder,
      scan,
      cancelScan,
      runExport,
      updateFilter,
      selectTag,
      selectRecord: setSelectedID,
      updateMoveDraft,
      browseMoveDestination,
      createMovePlan,
      openMoveDialog: () => reviewedMove && !isApplyingMove && setMoveDialogOpen(true),
      closeMoveDialog: () => setMoveDialogOpen(false),
      confirmMove
    }
  };
}

function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(timer);
  }, [delay, value]);
  return debounced;
}

function useWailsEvents(onProgress: (progress: ScanProgress) => void, onDrop: (path: string) => void) {
  const progressRef = useRef(onProgress);
  const dropRef = useRef(onDrop);
  progressRef.current = onProgress;
  dropRef.current = onDrop;

  useEffect(() => {
    const cancelProgress = window.runtime?.EventsOn?.("scan:progress", (item) => {
      progressRef.current(item as ScanProgress);
    });
    const cancelDrop = window.runtime?.EventsOn?.("wails:file-drop", (...args) => {
      const paths = extractDroppedPaths(args);
      if (paths.length > 0) dropRef.current(paths[0]);
    });
    return () => {
      cancelProgress?.();
      cancelDrop?.();
    };
  }, []);
}

function toSearchRequest(filters: Filters): SearchRequest {
  return {
    query: filters.query,
    tag: filters.tag,
    source: filters.source,
    format: filters.format,
    has_workflow: filters.hasWorkflow,
    limit: 200
  };
}

function extractDroppedPaths(args: unknown[]): string[] {
  for (const arg of args) {
    if (Array.isArray(arg) && arg.every((item) => typeof item === "string")) return arg as string[];
  }
  return [];
}

function messageFrom(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
