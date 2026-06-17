package mover

import (
	"context"
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
