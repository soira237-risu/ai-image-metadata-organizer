package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soira237-risu/ai-image-metadata-organizer/internal/appcore"
	"github.com/soira237-risu/ai-image-metadata-organizer/internal/store"
)

func TestScanCommandErrorHonorsFailOnError(t *testing.T) {
	result := appcore.ScanResult{Errors: []appcore.FileError{{Path: "bad.png", Error: "decode failed"}}}
	if err := scanCommandError(result, false); err != nil {
		t.Fatalf("default scan should preserve current exit behavior: %v", err)
	}
	err := scanCommandError(result, true)
	if err == nil || !strings.Contains(err.Error(), "1 file errors") {
		t.Fatalf("expected strict scan error, got %v", err)
	}
}

func TestRunWithIOContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dbPath := filepath.Join(t.TempDir(), "imv.db")

	err := runWithIOContext(ctx, []string{"stats", "--db", dbPath}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("expected canceled command, got %v", err)
	}
}

func TestParseInterspersedAllowsFlagsAfterPositional(t *testing.T) {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	db := fs.String("db", ".imv/imv.db", "")
	rescan := fs.Bool("rescan", false, "")

	err := parseInterspersed(fs, []string{"images", "--db", "custom.db", "--rescan"})
	if err != nil {
		t.Fatal(err)
	}
	if fs.NArg() != 1 || fs.Arg(0) != "images" {
		t.Fatalf("unexpected positional args: %v", fs.Args())
	}
	if *db != "custom.db" || !*rescan {
		t.Fatalf("unexpected flags: db=%q rescan=%v", *db, *rescan)
	}
}

func TestSubcommandHelpFlagPrintsCommandUsageToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"scan", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	for _, want := range []string{"Usage: imv scan", "-workers"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote stderr: %s", stderr.String())
	}
}

func TestHelpCommandPrintsSubcommandUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"help", "search"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: imv search") {
		t.Fatalf("unexpected help output:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote stderr: %s", stderr.String())
	}
}

func TestCommandsRejectUnexpectedPositionalArguments(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "imv.db")
	outPath := filepath.Join(tempDir, "out.json")
	tests := [][]string{
		{"search", "unexpected", "--db", dbPath},
		{"tags", "unexpected", "--db", dbPath},
		{"stats", "unexpected", "--db", dbPath},
		{"export", "unexpected", "--db", dbPath, "--out", outPath},
		{"move", "unexpected", "--db", dbPath, "--tag", "blue hair", "--to", tempDir},
	}

	for _, args := range tests {
		err := runWithIO(args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "usage: imv "+args[0]) {
			t.Fatalf("%v: expected usage error, got %v", args, err)
		}
	}
}

func TestCommandsRejectNonPositiveNumericOptions(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"scan", "images", "--workers", "0"}, want: "--workers must be greater than zero"},
		{args: []string{"search", "--limit", "0"}, want: "--limit must be greater than zero"},
		{args: []string{"tags", "--limit", "-1"}, want: "--limit must be greater than zero"},
	}

	for _, tt := range tests {
		err := runWithIO(tt.args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%v: expected %q, got %v", tt.args, tt.want, err)
		}
	}
}

func TestSearchTableIncludesReadableFields(t *testing.T) {
	dbPath := seedCLIDB(t)
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"search", "--db", dbPath, "--q", "blue"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("search failed: %v stderr=%s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"ID", "SOURCE", "FORMAT", "DIMENSIONS", "PROMPT", "TAGS", "nai", "png", "512x768", "1girl", "blue hair", "a.png"} {
		if !strings.Contains(output, want) {
			t.Fatalf("search output missing %q:\n%s", want, output)
		}
	}
}

func TestSearchJSONStillOutputsRecords(t *testing.T) {
	dbPath := seedCLIDB(t)
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"search", "--db", dbPath, "--format", "webp", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("search failed: %v stderr=%s", err, stderr.String())
	}
	var records []store.ImageRecord
	if err := json.Unmarshal(stdout.Bytes(), &records); err != nil {
		t.Fatalf("invalid json %v:\n%s", err, stdout.String())
	}
	if len(records) != 1 || records[0].Path != "b.webp" {
		t.Fatalf("unexpected json records: %#v", records)
	}
}

func TestTagsCommandOrdersByCount(t *testing.T) {
	dbPath := seedCLIDB(t)
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"tags", "--db", dbPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("tags failed: %v stderr=%s", err, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "TAG") || !strings.Contains(output, "COUNT") || !strings.Contains(output, "SOURCES") {
		t.Fatalf("tags output missing headers:\n%s", output)
	}
	if !strings.Contains(output, "blue hair") || !strings.Contains(output, "2") || !strings.Contains(output, "comfyui,nai") {
		t.Fatalf("tags output missing top tag:\n%s", output)
	}
}

func TestStatsJSONContainsAggregates(t *testing.T) {
	dbPath := seedCLIDB(t)
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"stats", "--db", dbPath, "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("stats failed: %v stderr=%s", err, stderr.String())
	}
	var stats store.Stats
	if err := json.Unmarshal(stdout.Bytes(), &stats); err != nil {
		t.Fatalf("invalid json %v:\n%s", err, stdout.String())
	}
	if stats.TotalFiles != 2 || stats.Formats["png"] != 1 || stats.Formats["webp"] != 1 {
		t.Fatalf("unexpected format stats: %#v", stats)
	}
	if stats.Sources["nai"] != 1 || stats.Sources["comfyui"] != 1 || stats.WorkflowFiles != 1 {
		t.Fatalf("unexpected source/workflow stats: %#v", stats)
	}
	if len(stats.TopTags) == 0 || stats.TopTags[0].Tag != "blue hair" || stats.TopTags[0].Count != 2 {
		t.Fatalf("unexpected top tags: %#v", stats.TopTags)
	}
}

func TestScanJSONOutput(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "sample.png")
	if err := writeMinimalPNG(imagePath); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "imv.db")
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"scan", dir, "--db", dbPath, "--workers", "1", "--json"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("scan failed: %v stderr=%s", err, stderr.String())
	}
	var result struct {
		Scanned int `json:"scanned"`
		Indexed int `json:"indexed"`
		Skipped int `json:"skipped"`
		Errors  []struct {
			Path  string `json:"path"`
			Error string `json:"error"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid json %v:\n%s", err, stdout.String())
	}
	if result.Scanned != 1 || result.Indexed != 1 || result.Skipped != 0 || len(result.Errors) != 0 {
		t.Fatalf("unexpected scan json: %#v", result)
	}
}

func TestShowHumanSummaryIncludesTagsSettingsAndWorkflow(t *testing.T) {
	dbPath := seedCLIDB(t)
	var stdout, stderr bytes.Buffer

	err := runWithIO([]string{"show", "b.webp", "--db", dbPath}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("show failed: %v stderr=%s", err, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{"tags=blue hair, landscape", "settings=sampler=euler", "workflow=node_count=4", "checkpoints=model.safetensors", "samplers=euler"} {
		if !strings.Contains(output, want) {
			t.Fatalf("show output missing %q:\n%s", want, output)
		}
	}
}

func seedCLIDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "imv.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	seedCLIRecord(t, db, store.FileInput{
		Path:   "a.png",
		Format: "png",
		Size:   100,
		MTime:  1,
		Width:  512,
		Height: 768,
		Metadata: []store.MetadataRecord{{
			Source:         "nai",
			PositivePrompt: "1girl, blue hair, smile",
			Settings:       map[string]any{"seed": float64(111)},
		}},
		Tags: []store.TagRecord{
			{Tag: "blue hair", Normalized: "blue hair", Source: "nai", Kind: "prompt"},
			{Tag: "smile", Normalized: "smile", Source: "nai", Kind: "prompt"},
		},
	})
	seedCLIRecord(t, db, store.FileInput{
		Path:   "b.webp",
		Format: "webp",
		Size:   200,
		MTime:  2,
		Width:  1024,
		Height: 1024,
		Metadata: []store.MetadataRecord{{
			Source:         "comfyui",
			PositivePrompt: "landscape, blue hair",
			Settings:       map[string]any{"sampler": "euler"},
			WorkflowSummary: map[string]any{
				"node_count":  float64(4),
				"checkpoints": []any{"model.safetensors"},
				"samplers":    []any{"euler"},
			},
		}},
		Tags: []store.TagRecord{
			{Tag: "landscape", Normalized: "landscape", Source: "comfyui", Kind: "prompt"},
			{Tag: "blue hair", Normalized: "blue hair", Source: "comfyui", Kind: "prompt"},
		},
	})
	return dbPath
}

func seedCLIRecord(t *testing.T, db *store.DB, input store.FileInput) {
	t.Helper()
	if _, err := db.UpsertFile(context.Background(), input); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalPNG(path string) error {
	var ihdr bytes.Buffer
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(1))
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(1))
	ihdr.Write([]byte{8, 2, 0, 0, 0})
	data := append([]byte{137, 80, 78, 71, 13, 10, 26, 10}, pngTestChunk("IHDR", ihdr.Bytes())...)
	data = append(data, pngTestChunk("IEND", nil)...)
	return os.WriteFile(path, data, 0644)
}

func pngTestChunk(kind string, data []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, uint32(len(data)))
	out.WriteString(kind)
	out.Write(data)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(data)
	_ = binary.Write(&out, binary.BigEndian, crc.Sum32())
	return out.Bytes()
}
