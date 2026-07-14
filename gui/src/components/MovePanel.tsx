import { FolderSearch, MoveRight, ShieldCheck, X } from "lucide-react";
import type { MoveDraft } from "../hooks/useLibraryController";
import type { LibraryController } from "./controller";

export function MovePanel({ controller }: { controller: LibraryController }) {
  const { moveDraft, reviewedMove, movePlanInvalidated, moveDialogOpen, isScanning, isPlanningMove, isApplyingMove, actions } = controller;
  const moveLocked = isScanning || isApplyingMove;
  const planCount = reviewedMove?.plans.length ?? 0;
  return (
    <section className="move-panel" aria-label="파일 정리">
      <div className="move-heading"><span className="move-icon"><MoveRight size={18} /></span><div><strong>태그로 정리</strong><small>계획을 확인한 뒤에만 실제 파일을 이동합니다.</small></div></div>
      <label>태그<input data-testid="move-tag-input" value={moveDraft.tag} disabled={moveLocked} onChange={(event) => actions.updateMoveDraft("tag", event.target.value)} placeholder="정리할 태그" /></label>
      <label className="destination-field">대상 폴더<span><input data-testid="move-to-input" value={moveDraft.to} disabled={moveLocked} onChange={(event) => actions.updateMoveDraft("to", event.target.value)} placeholder="대상 폴더 선택" /><button type="button" className="icon-button" disabled={moveLocked} onClick={() => void actions.browseMoveDestination()} title="대상 폴더 선택" aria-label="대상 폴더 선택"><FolderSearch size={16} /></button></span></label>
      <label>충돌<select value={moveDraft.conflict} disabled={moveLocked} onChange={(event) => actions.updateMoveDraft("conflict", event.target.value as MoveDraft["conflict"])}><option value="skip">건너뛰기</option><option value="rename">이름 바꾸기</option></select></label>
      <button className="button secondary" data-testid="plan-move-button" type="button" disabled={isPlanningMove || moveLocked} onClick={() => void actions.createMovePlan()}>{isPlanningMove ? "계획 중" : "계획"}</button>
      <button className="button primary" data-testid="apply-move-button" type="button" disabled={!reviewedMove || planCount === 0 || moveLocked} onClick={actions.openMoveDialog}>적용</button>
      <div className={movePlanInvalidated ? "plan-summary invalid" : "plan-summary"} data-testid="move-plan-summary">{movePlanInvalidated ? "입력이 바뀌어 다시 계획해야 합니다" : isPlanningMove ? "이동 계획을 확인하는 중…" : reviewedMove ? `${planCount} planned · 검토 완료` : "이동 계획 없음"}</div>
      {moveDialogOpen && reviewedMove && <div className="dialog-backdrop" role="presentation" onKeyDown={(event) => { if (event.key === "Escape") actions.closeMoveDialog(); }}><div className="confirm-dialog" role="dialog" aria-modal="true" aria-labelledby="move-dialog-title"><button className="dialog-close" type="button" onClick={actions.closeMoveDialog} aria-label="닫기"><X size={18} /></button><span className="confirm-icon"><ShieldCheck size={24} /></span><h2 id="move-dialog-title">파일 이동을 실행할까요?</h2><p><strong>{reviewedMove.plans.length}개</strong> 파일을 다음 위치로 이동합니다.</p><code>{reviewedMove.request.to}</code><small>충돌 처리: {reviewedMove.request.conflict === "rename" ? "자동 이름 변경" : "건너뛰기"}</small><div className="dialog-actions"><button className="button secondary" type="button" autoFocus onClick={actions.closeMoveDialog}>취소</button><button className="button primary" type="button" onClick={() => void actions.confirmMove()}>이동 실행</button></div></div></div>}
    </section>
  );
}
