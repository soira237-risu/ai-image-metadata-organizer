export type MetadataRecord = {
  source: string;
  positive_prompt?: string;
  negative_prompt?: string;
  settings?: Record<string, unknown>;
  workflow_summary?: Record<string, unknown>;
  raw?: Record<string, unknown>;
};

export type TagRecord = {
  tag: string;
  normalized_tag: string;
  source: string;
  kind: string;
};

export type ImageRecord = {
  id: number;
  path: string;
  format: string;
  size: number;
  mtime: number;
  width: number;
  height: number;
  scanned_at?: string;
  metadata?: MetadataRecord[];
  tags?: TagRecord[];
};

export type ImageDetail = {
  record: ImageRecord;
  preview_data_url?: string;
};

export type SearchRequest = {
  tag?: string;
  source?: string;
  query?: string;
  format?: string;
  has_workflow?: boolean;
  limit?: number;
};

export type TagsRequest = {
  source?: string;
  query?: string;
  limit?: number;
};

export type TagSummary = {
  tag: string;
  count: number;
  sources: string[];
  example: string;
};

export type Stats = {
  total_files: number;
  formats: Record<string, number>;
  sources: Record<string, number>;
  workflow_files: number;
  top_tags: TagSummary[];
  first_scanned_at?: string;
  last_scanned_at?: string;
};

export type ScanProgress = {
  path: string;
  total: number;
  done: number;
  scanned: number;
  indexed: number;
  skipped: number;
  errors: number;
};

export type ScanResult = {
  scanned: number;
  indexed: number;
  skipped: number;
  errors: Array<{ path: string; error: string }>;
};

export type MoveRequest = {
  tag: string;
  to: string;
  conflict?: "skip" | "rename";
};

export type MovePlan = {
  file_id: number;
  source_path: string;
  destination_path: string;
  reason: string;
  status: string;
};

export type FolderState = {
  folder: string;
  db_path: string;
};

export type ExportResult = {
  path: string;
  count: number;
};

export type BackendAPI = {
  OpenFolder(): Promise<FolderState>;
  ChooseDestinationFolder(): Promise<string>;
  State(): Promise<FolderState>;
  ScanFolder(folder: string, rescan: boolean): Promise<ScanResult>;
  CancelScan(): Promise<boolean>;
  Search(req: SearchRequest): Promise<ImageRecord[]>;
  GetImage(req: { id?: number; path?: string; ref?: string; include_raw?: boolean; include_preview?: boolean; preview_max_bytes?: number }): Promise<ImageDetail>;
  GetTags(req: TagsRequest): Promise<TagSummary[]>;
  GetStats(): Promise<Stats>;
  PlanMove(req: MoveRequest): Promise<MovePlan[]>;
  ApplyMove(req: MoveRequest): Promise<MovePlan[]>;
  ExportJSON(out: string, pretty: boolean): Promise<ExportResult>;
};

export type WailsRuntime = {
  EventsOn?: (eventName: string, callback: (...args: unknown[]) => void) => () => void;
};

declare global {
  interface Window {
    go?: {
      main?: {
        Backend?: BackendAPI;
      };
    };
    runtime?: WailsRuntime;
  }
}
