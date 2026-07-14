package mover

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Warning         string `json:"warning,omitempty"`
}

var errCrossDevice = errors.New("cross-device move")

type fileOps struct {
	rename     func(string, string) error
	stat       func(string) (os.FileInfo, error)
	open       func(string) (*os.File, error)
	createTemp func(string, string) (*os.File, error)
	remove     func(string) error
	mkdirAll   func(string, os.FileMode) error
	chmod      func(string, os.FileMode) error
	chtimes    func(string, time.Time, time.Time) error
}

func defaultFileOps() fileOps {
	return fileOps{
		rename:     os.Rename,
		stat:       os.Stat,
		open:       os.Open,
		createTemp: os.CreateTemp,
		remove:     os.Remove,
		mkdirAll:   os.MkdirAll,
		chmod:      os.Chmod,
		chtimes:    os.Chtimes,
	}
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
	records, err := db.RecordsByTag(ctx, opts.Tag)
	if err != nil {
		return nil, err
	}
	safeTag := SanitizePathSegment(opts.Tag)
	var plans []MovePlan
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
	name := value
	if ext := filepath.Ext(value); ext != "" {
		name = strings.TrimSuffix(value, ext)
	}
	if isWindowsReservedName(name) {
		value = "_" + value
	}
	return value
}

func isWindowsReservedName(value string) bool {
	upper := strings.ToUpper(strings.TrimSpace(value))
	switch upper {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) {
		return upper[3] >= '1' && upper[3] <= '9'
	}
	return false
}

func applyOne(ctx context.Context, db *store.DB, plan MovePlan, opts Options) (MovePlan, error) {
	return applyOneWithFS(ctx, db, plan, opts, defaultFileOps())
}

func applyOneWithFS(ctx context.Context, db *store.DB, plan MovePlan, opts Options, ops fileOps) (MovePlan, error) {
	dest := plan.DestinationPath
	if _, err := ops.stat(dest); err == nil {
		if opts.Conflict == "skip" {
			plan.Status = "skipped_conflict"
			return withLogWarning(plan, appendLog(opts.LogPath, plan)), nil
		}
		var renameErr error
		dest, renameErr = renamedPathWithFS(dest, ops)
		if renameErr != nil {
			return plan, renameErr
		}
		plan.DestinationPath = dest
	} else if !os.IsNotExist(err) {
		return plan, err
	}
	if err := ops.mkdirAll(filepath.Dir(dest), 0755); err != nil {
		return plan, err
	}
	if err := movePath(plan.SourcePath, dest, ops); err != nil {
		plan.Status = "failed"
		plan = withLogWarning(plan, appendLog(opts.LogPath, plan))
		return plan, err
	}
	if err := db.UpdatePath(ctx, plan.FileID, dest); err != nil {
		if rollbackErr := movePath(dest, plan.SourcePath, ops); rollbackErr == nil {
			plan.Status = "failed_rolled_back"
		} else {
			plan.Status = "failed_needs_manual_repair"
			err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		plan = withLogWarning(plan, appendLog(opts.LogPath, plan))
		return plan, err
	}
	plan.Status = "moved"
	return withLogWarning(plan, appendLog(opts.LogPath, plan)), nil
}

func renamedPathWithFS(path string, ops fileOps) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := ops.stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
}

func movePath(source, destination string, ops fileOps) error {
	if err := ops.rename(source, destination); err == nil {
		return nil
	} else if !errors.Is(err, errCrossDevice) && !isPlatformCrossDevice(err) {
		return err
	}
	return copyAcrossFilesystems(source, destination, ops)
}

func copyAcrossFilesystems(source, destination string, ops fileOps) (resultErr error) {
	sourceFile, err := ops.open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	temp, err := ops.createTemp(filepath.Dir(destination), ".imv-move-*")
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
	if _, err := io.Copy(temp, sourceFile); err != nil {
		return err
	}
	if err := sourceFile.Close(); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := ops.chmod(tempPath, info.Mode().Perm()); err != nil {
		return err
	}
	if _, err := ops.stat(destination); err == nil {
		return os.ErrExist
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := ops.rename(tempPath, destination); err != nil {
		return err
	}
	keepTemp = false
	if err := ops.chtimes(destination, info.ModTime(), info.ModTime()); err != nil {
		_ = ops.remove(destination)
		return err
	}
	if err := ops.remove(source); err != nil {
		_ = ops.remove(destination)
		return err
	}
	return nil
}

func withLogWarning(plan MovePlan, err error) MovePlan {
	if err != nil {
		plan.Warning = "move log: " + err.Error()
	}
	return plan
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
