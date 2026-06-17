package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"imv/internal/metadata"
	"imv/internal/store"
)

type Options struct {
	Root    string
	Workers int
	Rescan  bool
}

type Result struct {
	Scanned int
	Indexed int
	Skipped int
	Errors  []FileError
}

type FileError struct {
	Path string
	Err  error
}

func Scan(ctx context.Context, db *store.DB, opts Options) (Result, error) {
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	paths, err := discover(opts.Root)
	if err != nil {
		return Result{}, err
	}

	jobs := make(chan string)
	results := make(chan workerResult)
	var wg sync.WaitGroup
	for i := 0; i < opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- scanOne(ctx, db, path, opts.Rescan)
			}
		}()
	}

	go func() {
		for _, path := range paths {
			select {
			case <-ctx.Done():
				break
			case jobs <- path:
			}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	result := Result{Scanned: len(paths)}
	for item := range results {
		if item.Err != nil {
			result.Errors = append(result.Errors, FileError{Path: item.Path, Err: item.Err})
			continue
		}
		if item.Skipped {
			result.Skipped++
		} else {
			result.Indexed++
		}
	}
	return result, ctx.Err()
}

type workerResult struct {
	Path    string
	Skipped bool
	Err     error
}

func discover(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".png" || ext == ".webp" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func scanOne(ctx context.Context, db *store.DB, path string, rescan bool) workerResult {
	info, err := os.Stat(path)
	if err != nil {
		return workerResult{Path: path, Err: err}
	}
	mtime := info.ModTime().UnixNano()
	if !rescan {
		unchanged, err := db.IsUnchanged(path, info.Size(), mtime)
		if err != nil {
			return workerResult{Path: path, Err: err}
		}
		if unchanged {
			return workerResult{Path: path, Skipped: true}
		}
	}

	extracted, err := metadata.ExtractFile(path)
	if err != nil {
		return workerResult{Path: path, Err: fmt.Errorf("extract metadata: %w", err)}
	}
	input := store.FileInput{
		Path:     path,
		Format:   extracted.Format,
		Size:     info.Size(),
		MTime:    mtime,
		Width:    extracted.Width,
		Height:   extracted.Height,
		Metadata: convertMetadata(extracted.Metadata),
		Tags:     convertTags(extracted.Tags),
	}
	if _, err := db.UpsertFile(ctx, input); err != nil {
		return workerResult{Path: path, Err: err}
	}
	return workerResult{Path: path}
}

func convertMetadata(items []metadata.Record) []store.MetadataRecord {
	out := make([]store.MetadataRecord, 0, len(items))
	for _, item := range items {
		out = append(out, store.MetadataRecord{
			Source:          string(item.Source),
			PositivePrompt:  item.PositivePrompt,
			NegativePrompt:  item.NegativePrompt,
			Settings:        item.Settings,
			WorkflowSummary: item.WorkflowSummary,
			Raw:             item.Raw,
		})
	}
	return out
}

func convertTags(items []metadata.ImageTag) []store.TagRecord {
	out := make([]store.TagRecord, 0, len(items))
	for _, item := range items {
		out = append(out, store.TagRecord{
			Tag:        item.Value,
			Normalized: item.Normalized,
			Source:     string(item.Source),
			Kind:       item.Kind,
		})
	}
	return out
}
