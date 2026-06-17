package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestUpsertSearchExportAndUpdatePath(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "imv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := FileInput{
		Path:   "a.png",
		Format: "png",
		Size:   12,
		MTime:  34,
		Metadata: []MetadataRecord{{
			Source:         "nai",
			PositivePrompt: "1girl, blue hair",
			Settings:       map[string]any{"seed": float64(123)},
			Raw:            map[string]any{"Software": "NovelAI"},
		}},
		Tags: []TagRecord{{Tag: "blue hair", Normalized: "blue hair", Source: "nai", Kind: "prompt"}},
	}
	id, err := db.UpsertFile(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	unchanged, err := db.IsUnchanged("a.png", 12, 34)
	if err != nil {
		t.Fatal(err)
	}
	if !unchanged {
		t.Fatalf("expected unchanged record")
	}

	found, err := db.Search(SearchOptions{Tag: "Blue Hair", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Path != "a.png" {
		t.Fatalf("unexpected search result: %#v", found)
	}

	if err := db.UpdatePath(context.Background(), id, "sorted/blue hair/a.png"); err != nil {
		t.Fatal(err)
	}
	exported, err := db.Export()
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 1 || exported[0].Path != "sorted/blue hair/a.png" {
		t.Fatalf("unexpected export result: %#v", exported)
	}
}

func TestSplitPromptTagsNormalizesAndDeduplicates(t *testing.T) {
	tags := SplitPromptTags("1girl, blue hair, Blue Hair, ")
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %#v", tags)
	}
	if tags[0].Normalized != "1girl" || tags[1].Normalized != "blue hair" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestSearchFiltersTagsSummaryAndStats(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "imv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	seedStoreRecord(t, db, FileInput{
		Path:   "a.png",
		Format: "png",
		Size:   100,
		MTime:  1,
		Width:  512,
		Height: 768,
		Metadata: []MetadataRecord{{
			Source:         "nai",
			PositivePrompt: "1girl, blue hair, smile",
			Settings:       map[string]any{"seed": float64(111)},
		}},
		Tags: []TagRecord{
			{Tag: "blue hair", Normalized: "blue hair", Source: "nai", Kind: "prompt"},
			{Tag: "smile", Normalized: "smile", Source: "nai", Kind: "prompt"},
		},
	})
	seedStoreRecord(t, db, FileInput{
		Path:   "b.webp",
		Format: "webp",
		Size:   200,
		MTime:  2,
		Width:  1024,
		Height: 1024,
		Metadata: []MetadataRecord{{
			Source:         "comfyui",
			PositivePrompt: "landscape, blue hair",
			WorkflowSummary: map[string]any{
				"node_count":  float64(4),
				"checkpoints": []any{"model.safetensors"},
				"samplers":    []any{"euler"},
			},
		}},
		Tags: []TagRecord{
			{Tag: "landscape", Normalized: "landscape", Source: "comfyui", Kind: "prompt"},
			{Tag: "blue hair", Normalized: "blue hair", Source: "comfyui", Kind: "prompt"},
		},
	})

	webpRecords, err := db.Search(SearchOptions{Format: "webp", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(webpRecords) != 1 || webpRecords[0].Path != "b.webp" {
		t.Fatalf("unexpected format filter result: %#v", webpRecords)
	}

	workflowRecords, err := db.Search(SearchOptions{HasWorkflow: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(workflowRecords) != 1 || workflowRecords[0].Path != "b.webp" {
		t.Fatalf("unexpected workflow filter result: %#v", workflowRecords)
	}

	tags, err := db.TagsSummary(TagSummaryOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) < 1 || tags[0].Tag != "blue hair" || tags[0].Count != 2 {
		t.Fatalf("unexpected tag summary: %#v", tags)
	}
	if len(tags[0].Sources) != 2 || tags[0].Sources[0] != "comfyui" || tags[0].Sources[1] != "nai" {
		t.Fatalf("unexpected tag sources: %#v", tags[0].Sources)
	}

	stats, err := db.Stats()
	if err != nil {
		t.Fatal(err)
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
	if stats.FirstScanned == nil || stats.LastScanned == nil {
		t.Fatalf("expected scan time range: %#v", stats)
	}
}

func seedStoreRecord(t *testing.T, db *DB, input FileInput) {
	t.Helper()
	if _, err := db.UpsertFile(context.Background(), input); err != nil {
		t.Fatal(err)
	}
}
