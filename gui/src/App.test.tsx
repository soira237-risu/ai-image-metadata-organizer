import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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

function installBackend(overrides: Partial<BackendAPI> = {}) {
  const backend: BackendAPI = {
    OpenFolder: vi.fn().mockResolvedValue({ folder: "C:\\images", db_path: "C:\\images\\.imv\\imv.db" }),
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
  window.runtime = { EventsOn: vi.fn() };
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
    expect(within(detail).getByText("Positive")).toBeInTheDocument();
    expect(within(detail).getByText("1girl, blue hair, smile")).toBeInTheDocument();
  });

  test("plans and applies tag move after confirmation", async () => {
    const backend = installBackend();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<App />);

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "blue hair" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByText("Plan"));

    expect(await screen.findByText("1 planned")).toBeInTheDocument();
    expect(backend.PlanMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });

    fireEvent.click(screen.getByText("Apply"));

    await waitFor(() => {
      expect(window.confirm).toHaveBeenCalledWith("Move 1 files?");
      expect(backend.ApplyMove).toHaveBeenCalledWith({ tag: "blue hair", to: "D:\\sorted", conflict: "skip" });
    });
  });

  test("keeps rendering when Wails returns null for empty lists", async () => {
    installBackend({
      Search: vi.fn().mockResolvedValue(null as unknown as ImageRecord[]),
      GetTags: vi.fn().mockResolvedValue(null),
      PlanMove: vi.fn().mockResolvedValue(null)
    });
    render(<App />);

    expect(await screen.findByText("No indexed images")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("move-tag-input"), { target: { value: "missing" } });
    fireEvent.change(screen.getByTestId("move-to-input"), { target: { value: "D:\\sorted" } });
    fireEvent.click(screen.getByText("Plan"));

    expect(await screen.findByText("No move plan")).toBeInTheDocument();
  });
});
