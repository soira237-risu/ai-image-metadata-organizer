package scanner

import (
	"context"
	"path/filepath"
	"testing"

	"imv/internal/store"
)

func TestScanIndexesPNGFixtureAndSearchesTag(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "nai.png")
	if err := writePNGTextFixture(imagePath, map[string]string{
		"Software":    "NovelAI",
		"Description": "1girl, blue hair",
		"Comment":     `{"uc":"bad anatomy","seed":123,"sampler":"k_euler","scale":7}`,
	}); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(filepath.Join(dir, ".imv", "imv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	result, err := Scan(context.Background(), db, Options{Root: dir, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Indexed != 1 || result.Skipped != 0 || len(result.Errors) != 0 {
		t.Fatalf("unexpected scan result: %#v", result)
	}

	found, err := db.Search(store.SearchOptions{Tag: "blue hair", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Path != imagePath {
		t.Fatalf("unexpected search result: %#v", found)
	}
}

func TestScanIndexesWebPJSONPromptAndSearchesTag(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "nai.webp")
	if err := writeWebPJSONFixture(imagePath, `{"prompt":"city lights, rain","uc":"blur","seed":42}`); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(filepath.Join(dir, ".imv", "imv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	result, err := Scan(context.Background(), db, Options{Root: dir, Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Indexed != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected scan result: %#v", result)
	}

	found, err := db.Search(store.SearchOptions{Tag: "rain", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Path != imagePath {
		t.Fatalf("unexpected search result: %#v", found)
	}
}
