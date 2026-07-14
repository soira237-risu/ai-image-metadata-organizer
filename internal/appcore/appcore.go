package appcore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"imv/internal/mover"
	"imv/internal/scanner"
	"imv/internal/store"
)

const (
	DefaultDBPath          = ".imv/imv.db"
	DefaultPreviewMaxBytes = int64(25 * 1024 * 1024)
)

type Service struct {
	DBPath      string
	MoveLogPath string
}

type atomicFileOps struct {
	createTemp func(string, string) (*os.File, error)
	replace    func(string, string) error
	remove     func(string) error
	mkdirAll   func(string, os.FileMode) error
	chmod      func(string, os.FileMode) error
}

func defaultAtomicFileOps() atomicFileOps {
	return atomicFileOps{
		createTemp: os.CreateTemp,
		replace:    replaceFile,
		remove:     os.Remove,
		mkdirAll:   os.MkdirAll,
		chmod:      os.Chmod,
	}
}

type ScanRequest struct {
	Root    string `json:"root"`
	Workers int    `json:"workers"`
	Rescan  bool   `json:"rescan"`
}

type ScanProgress struct {
	Path    string `json:"path"`
	Total   int    `json:"total"`
	Done    int    `json:"done"`
	Scanned int    `json:"scanned"`
	Indexed int    `json:"indexed"`
	Skipped int    `json:"skipped"`
	Errors  int    `json:"errors"`
}

type ScanResult struct {
	Scanned int         `json:"scanned"`
	Indexed int         `json:"indexed"`
	Skipped int         `json:"skipped"`
	Errors  []FileError `json:"errors"`
}

type FileError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type SearchRequest struct {
	Tag         string `json:"tag"`
	Source      string `json:"source"`
	Query       string `json:"query"`
	Format      string `json:"format"`
	HasWorkflow bool   `json:"has_workflow"`
	Limit       int    `json:"limit"`
}

type GetImageRequest struct {
	Ref             string `json:"ref"`
	ID              int64  `json:"id"`
	Path            string `json:"path"`
	IncludeRaw      bool   `json:"include_raw"`
	IncludePreview  bool   `json:"include_preview"`
	PreviewMaxBytes int64  `json:"preview_max_bytes"`
}

type ImageDetail struct {
	Record         store.ImageRecord `json:"record"`
	PreviewDataURL string            `json:"preview_data_url,omitempty"`
}

type TagsRequest struct {
	Source string `json:"source"`
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
}

type ExportRequest struct {
	Out    string `json:"out"`
	Pretty bool   `json:"pretty"`
}

type MoveRequest struct {
	Tag      string `json:"tag"`
	To       string `json:"to"`
	Conflict string `json:"conflict"`
}

type MovePlan = mover.MovePlan

func New(dbPath string) *Service {
	if strings.TrimSpace(dbPath) == "" {
		dbPath = DefaultDBPath
	}
	return &Service{DBPath: dbPath}
}

func DBPathForFolder(folder string) string {
	if strings.TrimSpace(folder) == "" {
		return DefaultDBPath
	}
	return filepath.Join(folder, ".imv", "imv.db")
}

func (s *Service) Scan(ctx context.Context, req ScanRequest, progress func(ScanProgress)) (ScanResult, error) {
	if strings.TrimSpace(req.Root) == "" {
		return ScanResult{}, fmt.Errorf("scan root is required")
	}
	db, err := s.open()
	if err != nil {
		return ScanResult{}, err
	}
	defer db.Close()

	result, err := scanner.Scan(ctx, db, scanner.Options{
		Root:    req.Root,
		Workers: req.Workers,
		Rescan:  req.Rescan,
		OnProgress: func(item scanner.Progress) {
			if progress == nil {
				return
			}
			progress(ScanProgress{
				Path:    item.Path,
				Total:   item.Total,
				Done:    item.Done,
				Scanned: item.Scanned,
				Indexed: item.Indexed,
				Skipped: item.Skipped,
				Errors:  item.Errors,
			})
		},
	})
	if err != nil {
		return ScanResult{}, err
	}
	return convertScanResult(result), nil
}

func (s *Service) Search(ctx context.Context, req SearchRequest) ([]store.ImageRecord, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return db.Search(ctx, store.SearchOptions{
		Tag:         req.Tag,
		Source:      req.Source,
		Query:       req.Query,
		Format:      req.Format,
		HasWorkflow: req.HasWorkflow,
		Limit:       req.Limit,
	})
}

func (s *Service) GetImage(ctx context.Context, req GetImageRequest) (ImageDetail, error) {
	db, err := s.open()
	if err != nil {
		return ImageDetail{}, err
	}
	defer db.Close()
	record, err := getRecord(ctx, db, req)
	if err != nil {
		return ImageDetail{}, err
	}
	detail := ImageDetail{Record: record}
	if req.IncludePreview {
		limit := req.PreviewMaxBytes
		if limit <= 0 {
			limit = DefaultPreviewMaxBytes
		}
		preview, err := PreviewDataURL(record.Path, limit)
		if err != nil {
			return ImageDetail{}, err
		}
		detail.PreviewDataURL = preview
	}
	return detail, nil
}

func (s *Service) Tags(ctx context.Context, req TagsRequest) ([]store.TagSummary, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return db.TagsSummary(ctx, store.TagSummaryOptions{
		Source: req.Source,
		Query:  req.Query,
		Limit:  req.Limit,
	})
}

func (s *Service) Stats(ctx context.Context) (store.Stats, error) {
	db, err := s.open()
	if err != nil {
		return store.Stats{}, err
	}
	defer db.Close()
	return db.Stats(ctx)
}

func (s *Service) Export(ctx context.Context, req ExportRequest) ([]store.ImageRecord, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	records, err := db.Export(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Out) == "" {
		return records, nil
	}
	data, err := marshal(records, req.Pretty)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomically(req.Out, data, 0644); err != nil {
		return nil, err
	}
	return records, nil
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	return writeFileAtomicallyWithFS(path, data, mode, defaultAtomicFileOps())
}

func writeFileAtomicallyWithFS(path string, data []byte, mode os.FileMode, ops atomicFileOps) error {
	dir := filepath.Dir(path)
	if err := ops.mkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := ops.createTemp(dir, ".imv-export-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	keepTemp := true
	defer func() {
		_ = temp.Close()
		if keepTemp {
			_ = ops.remove(tempPath)
		}
	}()
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := ops.chmod(tempPath, mode); err != nil {
		return err
	}
	if err := ops.replace(tempPath, path); err != nil {
		return err
	}
	keepTemp = false
	return nil
}

func (s *Service) PlanMove(ctx context.Context, req MoveRequest) ([]MovePlan, error) {
	return s.move(ctx, req, false)
}

func (s *Service) ApplyMove(ctx context.Context, req MoveRequest) ([]MovePlan, error) {
	return s.move(ctx, req, true)
}

func PreviewDataURL(path string, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultPreviewMaxBytes
	}
	mime, err := imageMIME(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > maxBytes {
		return "", fmt.Errorf("preview file exceeds %d bytes", maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) move(ctx context.Context, req MoveRequest, apply bool) ([]MovePlan, error) {
	if strings.TrimSpace(req.Tag) == "" {
		return nil, fmt.Errorf("tag is required")
	}
	if strings.TrimSpace(req.To) == "" {
		return nil, fmt.Errorf("destination folder is required")
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return mover.PlanAndMaybeApply(ctx, db, mover.Options{
		Tag:      req.Tag,
		To:       req.To,
		Apply:    apply,
		Conflict: req.Conflict,
		LogPath:  s.moveLogPath(),
	})
}

func (s *Service) open() (*store.DB, error) {
	dbPath := s.DBPath
	if strings.TrimSpace(dbPath) == "" {
		dbPath = DefaultDBPath
	}
	return store.Open(dbPath)
}

func (s *Service) moveLogPath() string {
	if strings.TrimSpace(s.MoveLogPath) != "" {
		return s.MoveLogPath
	}
	dbPath := s.DBPath
	if strings.TrimSpace(dbPath) == "" {
		dbPath = DefaultDBPath
	}
	return filepath.Join(filepath.Dir(dbPath), "move-log.jsonl")
}

func convertScanResult(result scanner.Result) ScanResult {
	out := ScanResult{
		Scanned: result.Scanned,
		Indexed: result.Indexed,
		Skipped: result.Skipped,
	}
	for _, item := range result.Errors {
		out.Errors = append(out.Errors, FileError{Path: item.Path, Error: item.Err.Error()})
	}
	return out
}

func getRecord(ctx context.Context, db *store.DB, req GetImageRequest) (store.ImageRecord, error) {
	if req.ID > 0 {
		return db.GetByID(ctx, req.ID, req.IncludeRaw)
	}
	if strings.TrimSpace(req.Path) != "" {
		return db.GetByPath(ctx, req.Path, req.IncludeRaw)
	}
	if strings.TrimSpace(req.Ref) == "" {
		return store.ImageRecord{}, fmt.Errorf("image reference is required")
	}
	if id, err := strconv.ParseInt(req.Ref, 10, 64); err == nil {
		return db.GetByID(ctx, id, req.IncludeRaw)
	}
	return db.GetByPath(ctx, req.Ref, req.IncludeRaw)
}

func imageMIME(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", nil
	case ".webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported preview format %q", filepath.Ext(path))
	}
}

func marshal(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
