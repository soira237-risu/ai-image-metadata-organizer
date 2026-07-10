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
      { tag: "smile", normalized_tag: "smile", source: "nai", kind: "prompt" },
      { tag: "very long tag that should still be fully visible", normalized_tag: "very long tag that should still be fully visible", source: "nai", kind: "prompt" }
    ]
  }
];

function installBackend(overrides: Partial<BackendAPI> = {}) {
  const backend: BackendAPI = {
    OpenFolder: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db" }),
    OpenFile: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db", selected_path: "C:\\images\\a.png" }),
    Reset: vi.fn().mockResolvedValue({ folder: "", db_path: ".imv/imv.db" }),
    RevealFolder: vi.fn().mockResolvedValue(undefined),
    InspectFile: vi.fn().mockResolvedValue({ record: records[0], preview_data_url: "data:image/png;base64,AAAA" }),
    PreviewPath: vi.fn().mockResolvedValue("data:image/png;base64,BBBB"),
    State: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db" }),
    ScanFolder: vi.fn().mockResolvedValue({ scanned: 1, indexed: 1, skipped: 0, errors: [] }),
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
  window.runtime = { EventsOn: vi.fn(), OnFileDrop: vi.fn(), OnFileDropOff: vi.fn() };
  return backend;
}

beforeEach(() => {
  vi.restoreAllMocks();
  installBackend();
});

describe("App", () => {
  test("renders the public product identity in the header", async () => {
    render(<App />);

    expect(await screen.findByTitle("AI Image Metadata Organizer (IMV)")).toHaveTextContent("IMV");
  });

  test("sends search filters to the backend", async () => {
    const backend = installBackend();
    render(<App />);

    fireEvent.change(screen.getByTestId("query-input"), { target: { value: "blue" } });

    await waitFor(() => {
      expect(backend.Search).toHaveBeenCalledWith(expect.objectContaining({ query: "blue" }));
    });
  });

  test("opens files and folders from the Korean open menu", async () => {
    const backend = installBackend();
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "열기" }));
    fireEvent.click(screen.getByRole("button", { name: "파일 열기" }));

    await waitFor(() => {
      expect(backend.OpenFile).toHaveBeenCalled();
      expect(backend.InspectFile).toHaveBeenCalledWith("C:\\images\\a.png");
      expect(backend.ScanFolder).not.toHaveBeenCalledWith("C:\\images\\a.png", false);
    });

    fireEvent.click(screen.getByRole("button", { name: "열기" }));
    fireEvent.click(screen.getByRole("button", { name: "폴더 열기" }));

    await waitFor(() => {
      expect(backend.OpenFolder).toHaveBeenCalled();
    });
  });

  test("opens a single file without scanning and keeps its preview visible", async () => {
    const single = { ...records[0], id: 0, path: "C:\\single\\only.png" };
    const backend = installBackend({
      OpenFile: vi.fn().mockResolvedValue({
        folder: "C:\\single",
        db_path: "단일 파일 모드",
        selected_path: "C:\\single\\only.png"
      }),
      Search: vi.fn().mockResolvedValue([]),
      InspectFile: vi.fn().mockResolvedValue({
        record: single,
        preview_data_url: "data:image/png;base64,SINGLE"
      })
    });
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "열기" }));
    fireEvent.click(screen.getByRole("button", { name: "파일 열기" }));

    await waitFor(() => {
      expect(backend.InspectFile).toHaveBeenCalledWith("C:\\single\\only.png");
      expect(backend.ScanFolder).not.toHaveBeenCalled();
    });
    const detail = await screen.findByTestId("detail-panel");
    expect(within(detail).getByRole("img", { name: "only.png" })).toHaveAttribute("src", "data:image/png;base64,SINGLE");
    expect(screen.getByText("파일 불러옴: C:\\single\\only.png")).toBeInTheDocument();
  });

  test("loads a single dropped file path when Wails sends a string event payload", async () => {
    const events = new Map<string, (...args: unknown[]) => void>();
    const single = { ...records[0], id: 0, path: "C:\\drop\\dropped.webp", format: "webp" };
    const backend = installBackend({
      Search: vi.fn().mockResolvedValue([]),
      InspectFile: vi.fn().mockResolvedValue({
        record: single,
        preview_data_url: "data:image/webp;base64,DROP"
      })
    });
    window.runtime = {
      EventsOn: vi.fn((eventName, callback) => {
        events.set(eventName, callback);
      })
    };
    render(<App />);

    await waitFor(() => {
      expect(events.has("wails:file-drop")).toBe(true);
    });
    await act(async () => {
      events.get("wails:file-drop")?.("C:\\drop\\dropped.webp");
    });

    await waitFor(() => {
      expect(backend.InspectFile).toHaveBeenCalledWith("C:\\drop\\dropped.webp");
    });
    const detail = await screen.findByTestId("detail-panel");
    expect(within(detail).getByRole("img", { name: "dropped.webp" })).toHaveAttribute("src", "data:image/webp;base64,DROP");
  });

  test("registers Wails OnFileDrop and loads dropped image paths", async () => {
    let fileDropCallback: ((x: number, y: number, paths: string[]) => void) | undefined;
    const single = { ...records[0], id: 0, path: "C:\\drop\\native.png" };
    const backend = installBackend({
      Search: vi.fn().mockResolvedValue([]),
      InspectFile: vi.fn().mockResolvedValue({
        record: single,
        preview_data_url: "data:image/png;base64,NATIVE"
      })
    });
    window.runtime = {
      EventsOn: vi.fn(),
      OnFileDrop: vi.fn((callback) => {
        fileDropCallback = callback;
      }),
      OnFileDropOff: vi.fn()
    };
    const { unmount } = render(<App />);

    await waitFor(() => {
      expect(window.runtime?.OnFileDrop).toHaveBeenCalledWith(expect.any(Function), false);
    });

    await act(async () => {
      fileDropCallback?.(24, 48, ["C:\\drop\\native.png"]);
    });

    await waitFor(() => {
      expect(backend.InspectFile).toHaveBeenCalledWith("C:\\drop\\native.png");
    });
    const detail = await screen.findByTestId("detail-panel");
    expect(within(detail).getByRole("img", { name: "native.png" })).toHaveAttribute("src", "data:image/png;base64,NATIVE");

    unmount();
    expect(window.runtime?.OnFileDropOff).toHaveBeenCalled();
  });

  test("shows bottom loading progress while a file is being inspected", async () => {
    let resolveInspect: (detail: { record: ImageRecord; preview_data_url: string }) => void = () => undefined;
    const inspectPromise = new Promise<{ record: ImageRecord; preview_data_url: string }>((resolve) => {
      resolveInspect = resolve;
    });
    const single = { ...records[0], id: 0, path: "C:\\slow\\wait.png" };
    installBackend({
      OpenFile: vi.fn().mockResolvedValue({
        folder: "C:\\slow",
        db_path: "단일 파일 모드",
        selected_path: "C:\\slow\\wait.png"
      }),
      Search: vi.fn().mockResolvedValue([]),
      InspectFile: vi.fn().mockReturnValue(inspectPromise)
    });
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "열기" }));
    fireEvent.click(screen.getByRole("button", { name: "파일 열기" }));

    expect(await screen.findByText("파일 불러오는 중: C:\\slow\\wait.png")).toBeInTheDocument();
    expect(screen.getByText("0/1")).toBeInTheDocument();

    await act(async () => {
      resolveInspect({ record: single, preview_data_url: "data:image/png;base64,SLOW" });
      await inspectPromise;
    });

    expect(await screen.findByText("파일 불러옴: C:\\slow\\wait.png")).toBeInTheDocument();
    expect(screen.getByText("1/1")).toBeInTheDocument();
  });

  test("opens help inside the Korean options popup", async () => {
    render(<App />);

    fireEvent.click(screen.getByRole("button", { name: "옵션" }));

    const options = screen.getByRole("dialog", { name: "옵션" });
    expect(within(options).getByRole("button", { name: "도움말" })).toBeInTheDocument();
    fireEvent.click(within(options).getByRole("button", { name: "도움말" }));
    expect(within(options).getByText("파일 열기")).toBeInTheDocument();
    expect(within(options).getByText("단일 PNG/WebP 파일만 읽어서 바로 보여줍니다. 폴더 전체 스캔은 하지 않습니다.")).toBeInTheDocument();
    expect(within(options).getByText("이동 실행")).toBeInTheDocument();
  });

  test("applies display spacing and pane width settings from the options popup", async () => {
    render(<App />);

    const shell = screen.getByTestId("app-shell");
    fireEvent.click(screen.getByRole("button", { name: "옵션" }));
    const options = screen.getByRole("dialog", { name: "옵션" });

    fireEvent.change(within(options).getByLabelText("글자 크기"), { target: { value: "16" } });
    fireEvent.change(within(options).getByLabelText("간격"), { target: { value: "12" } });
    fireEvent.click(within(options).getByRole("button", { name: "레이아웃" }));
    fireEvent.change(within(options).getByLabelText("왼쪽 패널 폭"), { target: { value: "280" } });
    fireEvent.change(within(options).getByLabelText("상세 패널 폭"), { target: { value: "520" } });

    expect(shell).toHaveStyle("--ui-font-size: 16px");
    expect(shell).toHaveStyle("--ui-gap: 12px");
    expect(shell).toHaveStyle("--filter-width: 280px");
    expect(shell).toHaveStyle("--detail-width: 520px");
  });

  test("renders selected image detail", async () => {
    render(<App />);

    const detail = await screen.findByTestId("detail-panel");
    expect(within(detail).getByRole("heading", { name: "a.png" })).toBeInTheDocument();
    expect(within(detail).getByText("긍정 프롬프트")).toBeInTheDocument();
    expect(within(detail).getByText("1girl, blue hair, smile")).toBeInTheDocument();
    expect(within(detail).getByText("blue hair")).toHaveClass("tag-chip");
    expect(within(detail).getByText("very long tag that should still be fully visible")).toBeInTheDocument();
  });

  test("plans and applies tag move after confirmation", async () => {
    const backend = installBackend();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<App />);

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "blue hair" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByText("계획"));

    expect(await screen.findByText("1개 계획됨")).toBeInTheDocument();
    expect(backend.PlanMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });

    fireEvent.click(screen.getByText("이동 실행"));

    await waitFor(() => {
      expect(window.confirm).toHaveBeenCalledWith("1개 파일을 이동할까요?");
      expect(backend.ApplyMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });
    });
    const resultPanel = await screen.findByRole("region", { name: "이동 결과" });
    expect(resultPanel).toBeInTheDocument();
    expect(screen.getByText("이동됨 1")).toBeInTheDocument();
    await waitFor(() => {
      expect(within(resultPanel).getByAltText("a.png")).toHaveAttribute("src", "data:image/png;base64,BBBB");
    });
    fireEvent.click(screen.getByRole("button", { name: "이동 폴더 열기" }));
    expect(backend.RevealFolder).toHaveBeenCalledWith("D:\\sorted\\blue hair");
  });

  test("resets loaded state without deleting data", async () => {
    const backend = installBackend();
    render(<App />);

    expect(await screen.findByText("1개 결과")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "초기화" }));

    await waitFor(() => {
      expect(backend.Reset).toHaveBeenCalled();
      expect(screen.getByText("불러온 폴더 없음")).toBeInTheDocument();
      expect(screen.getByText("0개 결과")).toBeInTheDocument();
      expect(screen.getByText("선택 없음")).toBeInTheDocument();
    });
  });

  test("keeps rendering when Wails returns null for empty lists", async () => {
    installBackend({
      Search: vi.fn().mockResolvedValue(null as unknown as ImageRecord[]),
      GetTags: vi.fn().mockResolvedValue(null),
      PlanMove: vi.fn().mockResolvedValue(null)
    });
    render(<App />);

    expect(await screen.findByText("색인된 이미지가 없습니다")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "missing" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByText("계획"));

    expect(await screen.findByText("이동 계획 없음")).toBeInTheDocument();
  });
});
