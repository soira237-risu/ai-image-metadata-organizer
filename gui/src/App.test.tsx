import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import App from "./App";
import type { BackendAPI, ImageRecord } from "./types";

const records: ImageRecord[] = [
  {
    id: 1,
    path: "C:\\images\\a.png",
    format: "png",
    size: 2048,
    mtime: 1,
    width: 512,
    height: 768,
    metadata: [{
      source: "nai",
      positive_prompt: "1girl, blue hair, smile",
      negative_prompt: "bad anatomy",
      settings: { seed: 123 },
      workflow_summary: {}
    }],
    tags: [
      { tag: "blue hair", normalized_tag: "blue hair", source: "nai", kind: "prompt" },
      { tag: "smile", normalized_tag: "smile", source: "nai", kind: "prompt" }
    ]
  }
];

const secondRecord: ImageRecord = {
  ...records[0],
  id: 2,
  path: "C:\\images\\b.webp",
  format: "webp",
  metadata: [{ source: "comfyui", positive_prompt: "warm window light" }],
  tags: [{ tag: "warm light", normalized_tag: "warm light", source: "comfyui", kind: "prompt" }]
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve;
    reject = nextReject;
  });
  return { promise, resolve, reject };
}

function installBackend(overrides: Partial<BackendAPI> = {}) {
  const backend: BackendAPI = {
    OpenFolder: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db" }),
    ChooseDestinationFolder: vi.fn().mockResolvedValue("D:\\sorted"),
    State: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db" }),
    ScanFolder: vi.fn().mockResolvedValue({ scanned: 1, indexed: 1, skipped: 0, errors: [] }),
    CancelScan: vi.fn().mockResolvedValue(true),
    Search: vi.fn().mockResolvedValue(records),
    GetImage: vi.fn().mockResolvedValue({ record: records[0], preview_data_url: "data:image/png;base64,AAAA" }),
    GetTags: vi.fn().mockResolvedValue([{ tag: "blue hair", count: 1, sources: ["nai"], example: "blue hair" }]),
    GetStats: vi.fn().mockResolvedValue({
      total_files: 1,
      formats: { png: 1 },
      sources: { nai: 1 },
      workflow_files: 0,
      top_tags: []
    }),
    PlanMove: vi.fn().mockResolvedValue([{
      file_id: 1,
      source_path: "C:\\images\\a.png",
      destination_path: "D:\\sorted\\blue hair\\a.png",
      reason: "tag:blue hair",
      status: "planned"
    }]),
    ApplyMove: vi.fn().mockResolvedValue([{
      file_id: 1,
      source_path: "C:\\images\\a.png",
      destination_path: "D:\\sorted\\blue hair\\a.png",
      reason: "tag:blue hair",
      status: "moved"
    }]),
    ExportJSON: vi.fn().mockResolvedValue({ path: "C:\\images\\export.json", count: 1 }),
    ...overrides
  };
  window.go = { main: { Backend: backend } };
  window.runtime = { EventsOn: vi.fn().mockReturnValue(() => undefined) };
  return backend;
}

beforeEach(() => {
  vi.restoreAllMocks();
  installBackend();
});

describe("App", () => {
  test("sends search filters to the backend", async () => {
    const backend = installBackend();
    render(<App />);

    fireEvent.change(screen.getByTestId("query-input"), { target: { value: "blue" } });

    await waitFor(() => {
      expect(backend.Search).toHaveBeenCalledWith(expect.objectContaining({ query: "blue" }));
    });
  });

  test("renders selected image detail", async () => {
    render(<App />);

    const detail = await screen.findByTestId("detail-panel");
    expect(within(detail).getByRole("heading", { name: "a.png" })).toBeInTheDocument();
    expect(within(detail).getByText("긍정 프롬프트")).toBeInTheDocument();
    expect(within(detail).getByText("1girl, blue hair, smile")).toBeInTheDocument();
  });

  test("applies the exact reviewed move request through an in-app confirmation", async () => {
    const backend = installBackend();
    render(<App />);

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "blue hair" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByTestId("plan-move-button"));

    expect(await screen.findByText(/1.*planned/i)).toBeInTheDocument();
    expect(backend.PlanMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });

    fireEvent.click(screen.getByTestId("apply-move-button"));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "이동 실행" }));

    await waitFor(() => {
      expect(backend.ApplyMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });
      expect(screen.getByTestId("apply-move-button")).toBeDisabled();
    });
  });

  test("invalidates a reviewed move plan when its request changes", async () => {
    render(<App />);
    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "blue hair" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByTestId("plan-move-button"));

    await screen.findByText(/1.*planned/i);
    expect(screen.getByTestId("apply-move-button")).toBeEnabled();

    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "E:\\changed" } });
    expect(screen.getByTestId("apply-move-button")).toBeDisabled();
    expect(screen.getByText(/다시.*계획/)).toBeInTheDocument();
  });

  test("ignores a move plan response when the draft changed while planning", async () => {
    const pendingPlan = deferred<Awaited<ReturnType<BackendAPI["PlanMove"]>>>();
    installBackend({ PlanMove: vi.fn().mockReturnValue(pendingPlan.promise) });
    render(<App />);

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "blue hair" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByTestId("plan-move-button"));
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "E:\\changed" } });

    await act(async () => pendingPlan.resolve([{
      file_id: 1,
      source_path: "C:\\images\\a.png",
      destination_path: "D:\\sorted\\blue hair\\a.png",
      reason: "tag:blue hair",
      status: "planned"
    }]));

    expect(screen.getByTestId("apply-move-button")).toBeDisabled();
    expect(screen.getByText(/다시.*계획/)).toBeInTheDocument();
  });

  test("ignores an older search response that resolves after the newest one", async () => {
    const oldSearch = deferred<ImageRecord[]>();
    const newSearch = deferred<ImageRecord[]>();
    installBackend({
      Search: vi.fn((request) => {
        if (request.query === "old") return oldSearch.promise;
        if (request.query === "new") return newSearch.promise;
        return Promise.resolve([]);
      })
    });
    render(<App />);

    fireEvent.change(screen.getByTestId("query-input"), { target: { value: "old" } });
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 230));
    });
    fireEvent.change(screen.getByTestId("query-input"), { target: { value: "new" } });
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 230));
    });

    await act(async () => newSearch.resolve([secondRecord]));
    expect(await within(screen.getByTestId("record-list")).findByText("b.webp")).toBeInTheDocument();
    await act(async () => oldSearch.resolve(records));
    expect(within(screen.getByTestId("record-list")).queryByText("a.png")).not.toBeInTheDocument();
  });

  test("ignores an older detail response after a newer selection", async () => {
    const firstDetail = deferred<Awaited<ReturnType<BackendAPI["GetImage"]>>>();
    const secondDetail = deferred<Awaited<ReturnType<BackendAPI["GetImage"]>>>();
    installBackend({
      Search: vi.fn().mockResolvedValue([records[0], secondRecord]),
      GetImage: vi.fn((request) => request.id === 1 ? firstDetail.promise : secondDetail.promise)
    });
    render(<App />);

    fireEvent.click(await screen.findByRole("button", { name: /b\.webp/ }));
    await act(async () => secondDetail.resolve({ record: secondRecord }));
    expect(within(await screen.findByTestId("detail-panel")).getByRole("heading", { name: "b.webp" })).toBeInTheDocument();
    await act(async () => firstDetail.resolve({ record: records[0] }));
    expect(within(screen.getByTestId("detail-panel")).queryByRole("heading", { name: "a.png" })).not.toBeInTheDocument();
  });

  test("clears stale library state and loads the newly opened folder", async () => {
    const backend = installBackend({
      OpenFolder: vi.fn().mockResolvedValue({ folder: "D:\\new", db_path: "D:\\new\\.imv\\imv.db" }),
      Search: vi.fn()
        .mockResolvedValueOnce(records)
        .mockResolvedValueOnce([secondRecord])
    });
    render(<App />);
    expect(await within(screen.getByTestId("record-list")).findByText("a.png")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("open-folder-button"));

    expect(await within(screen.getByTestId("record-list")).findByText("b.webp")).toBeInTheDocument();
    expect(within(screen.getByTestId("record-list")).queryByText("a.png")).not.toBeInTheDocument();
    expect(backend.OpenFolder).toHaveBeenCalledTimes(1);
  });

  test("unsubscribes Wails events when the app unmounts", () => {
    const cancelProgress = vi.fn();
    const cancelDrop = vi.fn();
    window.runtime = {
      EventsOn: vi.fn()
        .mockReturnValueOnce(cancelProgress)
        .mockReturnValueOnce(cancelDrop)
    };
    const view = render(<App />);
    view.unmount();
    expect(cancelProgress).toHaveBeenCalledTimes(1);
    expect(cancelDrop).toHaveBeenCalledTimes(1);
  });

  test("cancels an active scan through the backend", async () => {
    const runningScan = deferred<Awaited<ReturnType<BackendAPI["ScanFolder"]>>>();
    const backend = installBackend({ ScanFolder: vi.fn().mockReturnValue(runningScan.promise) });
    render(<App />);

    fireEvent.click(await screen.findByTestId("scan-button"));
    fireEvent.click(await screen.findByTestId("cancel-scan-button"));

    await waitFor(() => expect(backend.CancelScan).toHaveBeenCalledTimes(1));
    await act(async () => runningScan.reject(new Error("context canceled")));
  });
});
