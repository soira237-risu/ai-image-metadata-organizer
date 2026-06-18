package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"imv/internal/appcore"
	"imv/internal/store"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Backend struct {
	ctx    context.Context
	mu     sync.Mutex
	folder string
	dbPath string
	file   string
}

type FolderState struct {
	Folder       string `json:"folder"`
	DBPath       string `json:"db_path"`
	SelectedPath string `json:"selected_path,omitempty"`
}

type ExportResult struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func NewBackend() *Backend {
	return &Backend{dbPath: appcore.DefaultDBPath}
}

func (b *Backend) startup(ctx context.Context) {
	b.ctx = ctx
}

func (b *Backend) OpenFolder() (FolderState, error) {
	folder, err := wailsruntime.OpenDirectoryDialog(b.ctx, wailsruntime.OpenDialogOptions{
		Title: "이미지 폴더 열기",
	})
	if err != nil {
		return FolderState{}, err
	}
	if strings.TrimSpace(folder) == "" {
		return b.State(), nil
	}
	return b.useFolder(folder), nil
}

func (b *Backend) OpenFile() (FolderState, error) {
	path, err := wailsruntime.OpenFileDialog(b.ctx, wailsruntime.OpenDialogOptions{
		Title: "이미지 파일 열기",
		Filters: []wailsruntime.FileFilter{{
			DisplayName: "PNG/WebP images (*.png;*.webp)",
			Pattern:     "*.png;*.webp",
		}},
	})
	if err != nil {
		return FolderState{}, err
	}
	if strings.TrimSpace(path) == "" {
		return b.State(), nil
	}
	return b.useFile(path), nil
}

func (b *Backend) Reset() FolderState {
	b.mu.Lock()
	b.folder = ""
	b.file = ""
	b.dbPath = appcore.DefaultDBPath
	b.mu.Unlock()
	return FolderState{DBPath: appcore.DefaultDBPath}
}

func (b *Backend) State() FolderState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return FolderState{Folder: b.folder, DBPath: b.dbPath, SelectedPath: b.file}
}

func (b *Backend) ScanFolder(folder string, rescan bool) (appcore.ScanResult, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		state := b.State()
		folder = state.Folder
	}
	if folder == "" {
		return appcore.ScanResult{}, fmt.Errorf("folder is required")
	}
	info, err := os.Stat(folder)
	if err != nil {
		return appcore.ScanResult{}, err
	}
	if !info.IsDir() {
		folder = filepath.Dir(folder)
	}
	state := b.useFolder(folder)
	service := appcore.New(state.DBPath)
	result, err := service.Scan(context.Background(), appcore.ScanRequest{
		Root:    state.Folder,
		Workers: 4,
		Rescan:  rescan,
	}, func(progress appcore.ScanProgress) {
		if b.ctx != nil {
			wailsruntime.EventsEmit(b.ctx, "scan:progress", progress)
		}
	})
	if b.ctx != nil {
		wailsruntime.EventsEmit(b.ctx, "scan:complete", result)
	}
	return result, err
}

func (b *Backend) Search(req appcore.SearchRequest) ([]store.ImageRecord, error) {
	return b.service().Search(context.Background(), req)
}

func (b *Backend) GetImage(req appcore.GetImageRequest) (appcore.ImageDetail, error) {
	if req.PreviewMaxBytes <= 0 {
		req.PreviewMaxBytes = appcore.DefaultPreviewMaxBytes
	}
	req.IncludePreview = true
	return b.service().GetImage(context.Background(), req)
}

func (b *Backend) InspectFile(path string) (appcore.ImageDetail, error) {
	return appcore.InspectFile(context.Background(), path, true, appcore.DefaultPreviewMaxBytes)
}

func (b *Backend) PreviewPath(path string) (string, error) {
	return appcore.PreviewDataURL(path, appcore.DefaultPreviewMaxBytes)
}

func (b *Backend) GetTags(req appcore.TagsRequest) ([]store.TagSummary, error) {
	return b.service().Tags(context.Background(), req)
}

func (b *Backend) GetStats() (store.Stats, error) {
	return b.service().Stats(context.Background())
}

func (b *Backend) PlanMove(req appcore.MoveRequest) ([]appcore.MovePlan, error) {
	return b.service().PlanMove(context.Background(), req)
}

func (b *Backend) ApplyMove(req appcore.MoveRequest) ([]appcore.MovePlan, error) {
	return b.service().ApplyMove(context.Background(), req)
}

func (b *Backend) ExportJSON(out string, pretty bool) (ExportResult, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		selected, err := wailsruntime.SaveFileDialog(b.ctx, wailsruntime.SaveDialogOptions{
			Title:           "Export metadata JSON",
			DefaultFilename: "imv-export.json",
			Filters: []wailsruntime.FileFilter{{
				DisplayName: "JSON files (*.json)",
				Pattern:     "*.json",
			}},
		})
		if err != nil {
			return ExportResult{}, err
		}
		out = selected
	}
	if out == "" {
		return ExportResult{}, nil
	}
	records, err := b.service().Export(context.Background(), appcore.ExportRequest{Out: out, Pretty: pretty})
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: out, Count: len(records)}, nil
}

func (b *Backend) RevealFolder(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("folder path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	name, args, err := revealCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return exec.Command(name, args...).Start()
}

func (b *Backend) useFolder(folder string) FolderState {
	state := FolderState{
		Folder: folder,
		DBPath: appcore.DBPathForFolder(folder),
	}
	b.mu.Lock()
	b.folder = state.Folder
	b.dbPath = state.DBPath
	b.file = ""
	b.mu.Unlock()
	return state
}

func (b *Backend) useFile(path string) FolderState {
	state := FolderState{
		Folder:       filepath.Dir(path),
		DBPath:       appcore.DBPathForFolder(filepath.Dir(path)),
		SelectedPath: path,
	}
	b.mu.Lock()
	b.folder = state.Folder
	b.dbPath = state.DBPath
	b.file = state.SelectedPath
	b.mu.Unlock()
	return state
}

func (b *Backend) service() *appcore.Service {
	b.mu.Lock()
	dbPath := b.dbPath
	b.mu.Unlock()
	return appcore.New(dbPath)
}

func revealCommand(goos, path string) (string, []string, error) {
	switch goos {
	case "windows":
		return "explorer", []string{path}, nil
	case "darwin":
		return "open", []string{path}, nil
	case "linux":
		return "xdg-open", []string{path}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q", goos)
	}
}
