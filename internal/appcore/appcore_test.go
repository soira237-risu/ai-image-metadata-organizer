package appcore

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"imv/internal/store"
)

func TestServiceScanSearchTagsStatsAndGetImage(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "nai.png")
	if err := writePNGTextFixture(imagePath, map[string]string{
		"Software":    "NovelAI",
		"Description": "1girl, blue hair",
		"Comment":     `{"uc":"bad anatomy","seed":123,"sampler":"k_euler","scale":7}`,
	}); err != nil {
		t.Fatal(err)
	}
	service := New(filepath.Join(dir, ".imv", "imv.db"))

	var progress []ScanProgress
	result, err := service.Scan(context.Background(), ScanRequest{Root: dir, Workers: 1}, func(item ScanProgress) {
		progress = append(progress, item)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 1 || result.Indexed != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected scan result: %#v", result)
	}
	if len(progress) != 1 || progress[0].Done != 1 || progress[0].Total != 1 {
		t.Fatalf("unexpected progress: %#v", progress)
	}

	records, err := service.Search(context.Background(), SearchRequest{Tag: "blue hair", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Path != imagePath {
		t.Fatalf("unexpected search records: %#v", records)
	}

	detail, err := service.GetImage(context.Background(), GetImageRequest{ID: records[0].ID, IncludeRaw: true, IncludePreview: true})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Record.Path != imagePath || !strings.HasPrefix(detail.PreviewDataURL, "data:image/png;base64,") {
		t.Fatalf("unexpected detail: %#v", detail)
	}
	if len(detail.Record.Metadata) != 1 || len(detail.Record.Metadata[0].Raw) == 0 {
		t.Fatalf("expected raw metadata in detail: %#v", detail.Record.Metadata)
	}

	tags, err := service.Tags(context.Background(), TagsRequest{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) == 0 || tags[0].Tag != "1girl" {
		t.Fatalf("unexpected tags: %#v", tags)
	}

	stats, err := service.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 || stats.Formats["png"] != 1 || stats.Sources["nai"] != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestInspectFileExtractsSingleImageWithoutDBScan(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "single.png")
	if err := writePNGTextFixture(imagePath, map[string]string{
		"Software":    "NovelAI",
		"Description": "solo, green eyes",
		"Comment":     `{"uc":"low quality","seed":456}`,
	}); err != nil {
		t.Fatal(err)
	}

	detail, err := InspectFile(context.Background(), imagePath, true, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Record.Path != imagePath || detail.Record.ID != 0 {
		t.Fatalf("unexpected record: %#v", detail.Record)
	}
	if !strings.HasPrefix(detail.PreviewDataURL, "data:image/png;base64,") {
		t.Fatalf("missing preview: %q", detail.PreviewDataURL)
	}
	if len(detail.Record.Tags) == 0 || detail.Record.Tags[0].Tag != "solo" {
		t.Fatalf("unexpected tags: %#v", detail.Record.Tags)
	}
	if _, err := os.Stat(filepath.Join(dir, ".imv", "imv.db")); !os.IsNotExist(err) {
		t.Fatalf("InspectFile should not create DB, err=%v", err)
	}
}

func TestServiceExportWritesStableJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, ".imv", "imv.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.UpsertFile(context.Background(), store.FileInput{
		Path:   filepath.Join(dir, "a.png"),
		Format: "png",
		Size:   1,
		MTime:  1,
		Tags:   []store.TagRecord{{Tag: "blue hair", Normalized: "blue hair", Source: "nai", Kind: "prompt"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "export", "data.json")
	service := New(dbPath)
	records, err := service.Export(context.Background(), ExportRequest{Out: out, Pretty: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("unexpected records: %#v", records)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []store.ImageRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid export json: %v\n%s", err, string(data))
	}
	if len(decoded) != 1 || decoded[0].Path != filepath.Join(dir, "a.png") {
		t.Fatalf("unexpected export: %#v", decoded)
	}
}

func TestServiceMoveDryRunAndApply(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("not real image"), 0644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, ".imv", "imv.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	id, err := db.UpsertFile(context.Background(), store.FileInput{
		Path:   source,
		Format: "png",
		Size:   14,
		MTime:  1,
		Tags:   []store.TagRecord{{Tag: "blue hair", Normalized: "blue hair", Source: "nai", Kind: "prompt"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	service := New(dbPath)
	destinationRoot := filepath.Join(dir, "sorted")

	dryRun, err := service.PlanMove(context.Background(), MoveRequest{Tag: "blue hair", To: destinationRoot})
	if err != nil {
		t.Fatal(err)
	}
	if len(dryRun) != 1 || dryRun[0].Status != "planned" {
		t.Fatalf("unexpected dry-run: %#v", dryRun)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source moved during dry-run: %v", err)
	}

	applied, err := service.ApplyMove(context.Background(), MoveRequest{Tag: "blue hair", To: destinationRoot})
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0].Status != "moved" {
		t.Fatalf("unexpected apply plan: %#v", applied)
	}
	if _, err := os.Stat(applied[0].DestinationPath); err != nil {
		t.Fatalf("destination missing: %v", err)
	}

	db, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	record, err := db.GetByID(id, false)
	if err != nil {
		t.Fatal(err)
	}
	if record.Path != applied[0].DestinationPath {
		t.Fatalf("DB path not updated: %q", record.Path)
	}
	records, err := service.Search(context.Background(), SearchRequest{Tag: "blue hair", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Path != applied[0].DestinationPath {
		t.Fatalf("search did not return moved path: %#v", records)
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".imv", "move-log.jsonl")); err != nil || len(data) == 0 {
		t.Fatalf("move log missing len=%d err=%v", len(data), err)
	}
}

func TestPreviewDataURLSupportsPNGWebPAndSizeCap(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "sample.png")
	if err := writeMinimalPNG(pngPath); err != nil {
		t.Fatal(err)
	}
	webpPath := filepath.Join(dir, "sample.webp")
	if err := os.WriteFile(webpPath, []byte("RIFF\x0c\x00\x00\x00WEBP"), 0644); err != nil {
		t.Fatal(err)
	}

	png, err := PreviewDataURL(pngPath, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(png, "data:image/png;base64,") {
		t.Fatalf("unexpected png data url: %q", png)
	}
	webp, err := PreviewDataURL(webpPath, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(webp, "data:image/webp;base64,") {
		t.Fatalf("unexpected webp data url: %q", webp)
	}
	if _, err := PreviewDataURL(pngPath, 1); err == nil {
		t.Fatal("expected size cap error")
	}
}

func writePNGTextFixture(path string, chunks map[string]string) error {
	var out bytes.Buffer
	out.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	out.Write(pngChunk("IHDR", buildIHDR(1, 1)))
	for key, value := range chunks {
		data := append(append([]byte{}, []byte(key)...), append([]byte{0}, []byte(value)...)...)
		out.Write(pngChunk("tEXt", data))
	}
	out.Write(pngChunk("IEND", nil))
	return os.WriteFile(path, out.Bytes(), 0644)
}

func writeMinimalPNG(path string) error {
	var out bytes.Buffer
	out.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	out.Write(pngChunk("IHDR", buildIHDR(1, 1)))
	out.Write(pngChunk("IEND", nil))
	return os.WriteFile(path, out.Bytes(), 0644)
}

func buildIHDR(width, height uint32) []byte {
	var data bytes.Buffer
	_ = binary.Write(&data, binary.BigEndian, width)
	_ = binary.Write(&data, binary.BigEndian, height)
	data.Write([]byte{8, 2, 0, 0, 0})
	return data.Bytes()
}

func pngChunk(kind string, data []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, uint32(len(data)))
	out.WriteString(kind)
	out.Write(data)
	crc := crc32.ChecksumIEEE(append([]byte(kind), data...))
	_ = binary.Write(&out, binary.BigEndian, crc)
	return out.Bytes()
}
