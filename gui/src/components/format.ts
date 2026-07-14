import type { ImageRecord } from "../types";

export function recordSource(record: ImageRecord): string {
  return record.metadata?.[0]?.source || "-";
}

export function dimensions(record: ImageRecord): string {
  return record.width > 0 && record.height > 0 ? `${record.width} × ${record.height}` : "-";
}

export function promptPreview(record: ImageRecord): string {
  const prompt = record.metadata?.find((meta) => meta.positive_prompt)?.positive_prompt ?? "";
  return truncate(prompt.replace(/\s+/g, " "), 120);
}

export function tagPreview(record: ImageRecord, max = 8): string {
  return (record.tags ?? []).slice(0, max).map((tag) => tag.tag || tag.normalized_tag).filter(Boolean).join(", ");
}

export function fileName(path: string): string {
  return path.split(/[\\/]/).pop() || path;
}

export function shortPath(path: string): string {
  if (path.length <= 72) return path;
  return `…${path.slice(path.length - 70)}`;
}

export function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export function countMap(value: Record<string, number>): string {
  return Object.entries(value).sort(([a], [b]) => a.localeCompare(b)).map(([key, count]) => `${key.toUpperCase()} ${count}`).join(" · ");
}

function truncate(value: string, max: number): string {
  return value.length <= max ? value : `${value.slice(0, max - 1)}…`;
}
