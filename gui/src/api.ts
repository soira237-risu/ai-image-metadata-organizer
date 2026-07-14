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
    throw new Error("Wails backend is not available");
  }
  return api;
}

export function openFolder(): Promise<FolderState> {
  return backend().OpenFolder();
}

export function chooseDestinationFolder(): Promise<string> {
  return backend().ChooseDestinationFolder();
}

export function getState(): Promise<FolderState> {
  return backend().State();
}

export function scanFolder(folder: string, rescan: boolean): Promise<ScanResult> {
  return backend().ScanFolder(folder, rescan);
}

export function cancelScan(): Promise<boolean> {
  return backend().CancelScan();
}

export function search(req: SearchRequest): Promise<ImageRecord[]> {
  return backend().Search(req);
}

export function getImage(id: number): Promise<ImageDetail> {
  return backend().GetImage({ id, include_raw: true, include_preview: true });
}

export function getTags(req: TagsRequest): Promise<TagSummary[]> {
  return backend().GetTags(req);
}

export function getStats(): Promise<Stats> {
  return backend().GetStats();
}

export function planMove(req: MoveRequest): Promise<MovePlan[]> {
  return backend().PlanMove(req);
}

export function applyMove(req: MoveRequest): Promise<MovePlan[]> {
  return backend().ApplyMove(req);
}

export function exportJSON(pretty = true): Promise<ExportResult> {
  return backend().ExportJSON("", pretty);
}
