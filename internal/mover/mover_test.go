package mover

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"imv/internal/store"
)

func TestSanitizePathSegmentReplacesInvalidCharacters(t *testing.T) {
	got := SanitizePathSegment(` bad:/\*tag? `)
	if got != "bad____tag_" {
		t.Fatalf("unexpected sanitized segment %q", got)
	}
}

func TestPlanDoesNotMoveWithoutApply(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("not real image"), 0644); err != nil {
		t.Fatal(err)
	}
	db, id := seedMoveRecord(t, dir, source)

	plans, err := PlanAndMaybeApply(context.Background(), db, Options{
		Tag:     "blue hair",
		To:      filepath.Join(dir, "sorted"),
		LogPath: filepath.Join(dir, ".imv", "move-log.jsonl"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Status != "planned" {
		t.Fatalf("unexpected plans: %#v", plans)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source should remain in place for dry-run: %v", err)
	}
	record, err := db.GetByID(id, false)
	if err != nil {
		t.Fatal(err)
	}
	if record.Path != source {
		t.Fatalf("DB path changed during dry-run: %q", record.Path)
	}
}

func TestApplyMovesFileUpdatesDBAndWritesLog(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("not real image"), 0644); err != nil {
		t.Fatal(err)
	}
	db, id := seedMoveRecord(t, dir, source)
	logPath := filepath.Join(dir, ".imv", "move-log.jsonl")

	plans, err := PlanAndMaybeApply(context.Background(), db, Options{
		Tag:     "blue hair",
		To:      filepath.Join(dir, "sorted"),
		Apply:   true,
		LogPath: logPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Status != "moved" {
		t.Fatalf("unexpected plans: %#v", plans)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source should be moved, stat err=%v", err)
	}
	if _, err := os.Stat(plans[0].DestinationPath); err != nil {
		t.Fatalf("destination should exist: %v", err)
	}
	record, err := db.GetByID(id, false)
	if err != nil {
		t.Fatal(err)
	}
	if record.Path != plans[0].DestinationPath {
		t.Fatalf("DB path not updated: %q", record.Path)
	}
	if data, err := os.ReadFile(logPath); err != nil || len(data) == 0 {
		t.Fatalf("move log not written: len=%d err=%v", len(data), err)
	}
}

func TestApplySkipConflictKeepsSourceAndLogsStatus(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	db, id := seedMoveRecord(t, dir, source)
	destination := filepath.Join(dir, "sorted", "blue hair", "image.png")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, ".imv", "move-log.jsonl")

	plans, err := PlanAndMaybeApply(context.Background(), db, Options{
		Tag:      "blue hair",
		To:       filepath.Join(dir, "sorted"),
		Apply:    true,
		Conflict: "skip",
		LogPath:  logPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Status != "skipped_conflict" {
		t.Fatalf("unexpected plans: %#v", plans)
	}
	if data, err := os.ReadFile(source); err != nil || string(data) != "source" {
		t.Fatalf("source should remain unchanged, data=%q err=%v", data, err)
	}
	record, err := db.GetByID(id, false)
	if err != nil {
		t.Fatal(err)
	}
	if record.Path != source {
		t.Fatalf("DB path should remain unchanged: %q", record.Path)
	}
	assertMoveLogStatus(t, logPath, "skipped_conflict")
}

func TestApplyRenameConflictMovesWithNumericSuffix(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	db, id := seedMoveRecord(t, dir, source)
	existing := filepath.Join(dir, "sorted", "blue hair", "image.png")
	if err := os.MkdirAll(filepath.Dir(existing), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	plans, err := PlanAndMaybeApply(context.Background(), db, Options{
		Tag:      "blue hair",
		To:       filepath.Join(dir, "sorted"),
		Apply:    true,
		Conflict: "rename",
		LogPath:  filepath.Join(dir, ".imv", "move-log.jsonl"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "sorted", "blue hair", "image-1.png")
	if len(plans) != 1 || plans[0].Status != "moved" || plans[0].DestinationPath != want {
		t.Fatalf("unexpected plans: %#v want %q", plans, want)
	}
	if data, err := os.ReadFile(want); err != nil || string(data) != "source" {
		t.Fatalf("renamed destination mismatch, data=%q err=%v", data, err)
	}
	record, err := db.GetByID(id, false)
	if err != nil {
		t.Fatal(err)
	}
	if record.Path != want {
		t.Fatalf("DB path not updated to renamed path: %q", record.Path)
	}
}

func TestApplyRollsBackFileMoveWhenDBUpdateFails(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("not real image"), 0644); err != nil {
		t.Fatal(err)
	}
	db, id := seedMoveRecord(t, dir, source)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	plan := MovePlan{
		FileID:          id,
		SourcePath:      source,
		DestinationPath: filepath.Join(dir, "sorted", "blue hair", "image.png"),
		Reason:          "tag:blue hair",
		Status:          "planned",
	}
	applied, err := applyOne(context.Background(), db, plan, Options{
		Conflict: "skip",
		LogPath:  filepath.Join(dir, ".imv", "move-log.jsonl"),
	})
	if err == nil {
		t.Fatal("expected DB update failure")
	}
	if applied.Status != "failed_rolled_back" {
		t.Fatalf("status = %q", applied.Status)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source should be restored after rollback: %v", err)
	}
	if _, err := os.Stat(plan.DestinationPath); !os.IsNotExist(err) {
		t.Fatalf("destination should not remain after rollback, err=%v", err)
	}
}

func seedMoveRecord(t *testing.T, dir, source string) (*store.DB, int64) {
	t.Helper()
	db, err := store.Open(filepath.Join(dir, ".imv", "imv.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	id, err := db.UpsertFile(context.Background(), store.FileInput{
		Path:   source,
		Format: "png",
		Size:   14,
		MTime:  1,
		Tags: []store.TagRecord{{
			Tag:        "blue hair",
			Normalized: "blue hair",
			Source:     "nai",
			Kind:       "prompt",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, id
}

func assertMoveLogStatus(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entry struct {
		Plan MovePlan `json:"plan"`
	}
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("invalid move log json: %v\n%s", err, string(data))
	}
	if entry.Plan.Status != want {
		t.Fatalf("move log status = %q, want %q", entry.Plan.Status, want)
	}
}
