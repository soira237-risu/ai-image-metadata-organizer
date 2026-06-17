import type {
  BackendAPI,
  ExportResult,
  FolderState,
  ImageDetail,
  ImageRecord,
  MovePlan,
  MoveRequest,
  ScanResult,
  SearchRequest,
  Stats,
  TagSummary,
  TagsRequest
} from "./types";

function backend(): BackendAPI {
  const api = window.go?.main?.Backend;
  if (!api) {
    return unavailableBackend;
  }
  return api;
}

const unavailableBackend: BackendAPI = {
  OpenFolder: unavailable,
  State: unavailable,
  ScanFolder: unavailable,
  Search: unavailable,
  GetImage: unavailable,
  GetTags: unavailable,
  GetStats: unavailable,
  PlanMove: unavailable,
  ApplyMove: unavailable,
  ExportJSON: unavailable
};

function unavailable(): Promise<never> {
  return Promise.reject(new Error("Wails backend is not available"));
}

export function openFolder(): Promise<FolderState> {
  return backend().OpenFolder();
}

export function getState(): Promise<FolderState> {
  return backend().State();
}

export function scanFolder(folder: string, rescan: boolean): Promise<ScanResult> {
  return backend().ScanFolder(folder, rescan).then((result) => ({
    ...result,
    errors: asArray(result?.errors)
  }));
}

export function search(req: SearchRequest): Promise<ImageRecord[]> {
  return backend().Search(req).then(asArray);
}

export function getImage(id: number): Promise<ImageDetail> {
  return backend().GetImage({ id, include_raw: true, include_preview: true });
}

export function getTags(req: TagsRequest): Promise<TagSummary[]> {
  return backend().GetTags(req).then(asArray);
}

export function getStats(): Promise<Stats> {
  return backend().GetStats().then((stats) => ({
    ...stats,
    formats: stats?.formats ?? {},
    sources: stats?.sources ?? {},
    top_tags: asArray(stats?.top_tags)
  }));
}

export function planMove(req: MoveRequest): Promise<MovePlan[]> {
  return backend().PlanMove(req).then(asArray);
}

export function applyMove(req: MoveRequest): Promise<MovePlan[]> {
  return backend().ApplyMove(req).then(asArray);
}

export function exportJSON(pretty = true): Promise<ExportResult> {
  return backend().ExportJSON("", pretty);
}

function asArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}
