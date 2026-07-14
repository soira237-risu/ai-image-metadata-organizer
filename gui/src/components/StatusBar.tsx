import { CircleCheck, Database } from "lucide-react";
import type { LibraryController } from "./controller";

export function StatusBar({ controller }: { controller: LibraryController }) {
  const { status, progress, folderState, isScanning } = controller;
  const percent = progress?.total ? Math.round((progress.done / progress.total) * 100) : 0;
  return <footer className="statusbar"><span className="status-message">{isScanning ? <Database className="pulse" size={14} /> : <CircleCheck size={14} />}{status}</span>{isScanning && progress && <span className="progress-wrap"><span className="progress-track"><i style={{ width: `${percent}%` }} /></span><strong>{progress.done}/{progress.total}</strong></span>}<span className="db-status"><i />{folderState.folder ? "로컬 DB 연결됨" : "폴더 대기 중"}</span></footer>;
}
