package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/soira237-risu/ai-image-metadata-organizer/internal/appcore"
	"github.com/soira237-risu/ai-image-metadata-organizer/internal/store"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Backend struct {
	ctx    context.Context
	mu     sync.Mutex
	folder string
	dbPath string

	scanCancel context.CancelFunc
	scanSeq    uint64
}

type FolderState struct {
	Folder string `json:"folder"`
	DBPath string `json:"db_path"`
}

type ExportResult struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func NewBackend() *Backend {
	return &Backend{dbPath: appcore.DefaultDBPath}
}

func (b *Backend) startup(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ctx = ctx
}

func (b *Backend) shutdown(context.Context) {
	b.CancelScan()
}

func (b *Backend) OpenFolder() (FolderState, error) {
	folder, err := wailsruntime.OpenDirectoryDialog(b.appContext(), wailsruntime.OpenDialogOptions{
		Title: "Open image folder",
	})
	if err != nil {
		return FolderState{}, err
	}
	if strings.TrimSpace(folder) == "" {
		return b.State(), nil
	}
	return b.useFolder(folder), nil
}

func (b *Backend) ChooseDestinationFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(b.appContext(), wailsruntime.OpenDialogOptions{
		Title: "Choose destination folder",
	})
}

func (b *Backend) State() FolderState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return FolderState{Folder: b.folder, DBPath: b.dbPath}
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
	scanCtx, finish := b.beginScanContext()
	defer finish()
	appCtx := b.appContext()
	result, err := service.Scan(scanCtx, appcore.ScanRequest{
		Root:    state.Folder,
		Workers: 4,
		Rescan:  rescan,
	}, func(progress appcore.ScanProgress) {
		wailsruntime.EventsEmit(appCtx, "scan:progress", progress)
	})
	wailsruntime.EventsEmit(appCtx, "scan:complete", result)
	return result, err
}

func (b *Backend) Search(req appcore.SearchRequest) ([]store.ImageRecord, error) {
	return b.service().Search(b.appContext(), req)
}

func (b *Backend) GetImage(req appcore.GetImageRequest) (appcore.ImageDetail, error) {
	if req.PreviewMaxBytes <= 0 {
		req.PreviewMaxBytes = appcore.DefaultPreviewMaxBytes
	}
	req.IncludePreview = true
	return b.service().GetImage(b.appContext(), req)
}

func (b *Backend) GetTags(req appcore.TagsRequest) ([]store.TagSummary, error) {
	return b.service().Tags(b.appContext(), req)
}

func (b *Backend) GetStats() (store.Stats, error) {
	return b.service().Stats(b.appContext())
}

func (b *Backend) PlanMove(req appcore.MoveRequest) ([]appcore.MovePlan, error) {
	return b.service().PlanMove(b.appContext(), req)
}

func (b *Backend) ApplyMove(req appcore.MoveRequest) ([]appcore.MovePlan, error) {
	return b.service().ApplyMove(b.appContext(), req)
}

func (b *Backend) ExportJSON(out string, pretty bool) (ExportResult, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		selected, err := wailsruntime.SaveFileDialog(b.appContext(), wailsruntime.SaveDialogOptions{
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
	records, err := b.service().Export(b.appContext(), appcore.ExportRequest{Out: out, Pretty: pretty})
	if err != nil {
		return ExportResult{}, err
	}
	return ExportResult{Path: out, Count: len(records)}, nil
}

func (b *Backend) useFolder(folder string) FolderState {
	state := FolderState{
		Folder: folder,
		DBPath: appcore.DBPathForFolder(folder),
	}
	b.mu.Lock()
	b.folder = state.Folder
	b.dbPath = state.DBPath
	b.mu.Unlock()
	return state
}

func (b *Backend) service() *appcore.Service {
	b.mu.Lock()
	dbPath := b.dbPath
	b.mu.Unlock()
	return appcore.New(dbPath)
}

func (b *Backend) appContext() context.Context {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctx != nil {
		return b.ctx
	}
	return context.Background()
}

func (b *Backend) beginScanContext() (context.Context, func()) {
	base := b.appContext()
	b.mu.Lock()
	if b.scanCancel != nil {
		b.scanCancel()
	}
	b.scanSeq++
	seq := b.scanSeq
	ctx, cancel := context.WithCancel(base)
	b.scanCancel = cancel
	b.mu.Unlock()
	return ctx, func() {
		cancel()
		b.mu.Lock()
		if b.scanSeq == seq {
			b.scanCancel = nil
		}
		b.mu.Unlock()
	}
}

func (b *Backend) CancelScan() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.scanCancel == nil {
		return false
	}
	b.scanCancel()
	return true
}
