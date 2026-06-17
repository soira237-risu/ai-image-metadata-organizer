package mover

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"imv/internal/store"
)

type Options struct {
	Tag      string
	To       string
	Apply    bool
	Conflict string
	LogPath  string
}

type MovePlan struct {
	FileID          int64  `json:"file_id"`
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	Reason          string `json:"reason"`
	Status          string `json:"status"`
}

func PlanAndMaybeApply(ctx context.Context, db *store.DB, opts Options) ([]MovePlan, error) {
	if opts.Conflict == "" {
		opts.Conflict = "skip"
	}
	if opts.Conflict != "skip" && opts.Conflict != "rename" {
		return nil, fmt.Errorf("unsupported conflict behavior %q", opts.Conflict)
	}
	if opts.LogPath == "" {
		opts.LogPath = ".imv/move-log.jsonl"
	}
	records, err := db.RecordsByTag(opts.Tag)
	if err != nil {
		return nil, err
	}
	safeTag := SanitizePathSegment(opts.Tag)
	plans := make([]MovePlan, 0, len(records))
	for _, record := range records {
		dest := filepath.Join(opts.To, safeTag, filepath.Base(record.Path))
		plan := MovePlan{
			FileID:          record.ID,
			SourcePath:      record.Path,
			DestinationPath: dest,
			Reason:          "tag:" + store.NormalizeTag(opts.Tag),
			Status:          "planned",
		}
		if !opts.Apply {
			plans = append(plans, plan)
			continue
		}
		applied, err := applyOne(ctx, db, plan, opts)
		if err != nil {
			return nil, err
		}
		plans = append(plans, applied)
	}
	return plans, nil
}

func SanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "untagged"
	}
	replacer := strings.NewReplacer("<", "_", ">", "_", ":", "_", `"`, "_", "/", "_", `\`, "_", "|", "_", "?", "_", "*", "_")
	value = replacer.Replace(value)
	value = strings.Trim(value, " .")
	if value == "" {
		return "untagged"
	}
	return value
}

func applyOne(ctx context.Context, db *store.DB, plan MovePlan, opts Options) (MovePlan, error) {
	dest := plan.DestinationPath
	if _, err := os.Stat(dest); err == nil {
		if opts.Conflict == "skip" {
			plan.Status = "skipped_conflict"
			return plan, appendLog(opts.LogPath, plan)
		}
		dest = renamedPath(dest)
		plan.DestinationPath = dest
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return plan, err
	}
	if err := os.Rename(plan.SourcePath, dest); err != nil {
		plan.Status = "failed"
		_ = appendLog(opts.LogPath, plan)
		return plan, err
	}
	if err := db.UpdatePath(ctx, plan.FileID, dest); err != nil {
		if rollbackErr := os.Rename(dest, plan.SourcePath); rollbackErr == nil {
			plan.Status = "failed_rolled_back"
		} else {
			plan.Status = "failed_needs_manual_repair"
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		_ = appendLog(opts.LogPath, plan)
		return plan, err
	}
	plan.Status = "moved"
	return plan, appendLog(opts.LogPath, plan)
}

func renamedPath(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func appendLog(path string, plan MovePlan) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	entry := map[string]any{
		"at":   time.Now().UTC().Format(time.RFC3339Nano),
		"plan": plan,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}
