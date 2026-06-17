package metadata

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"strings"
	"testing"
)

func TestParsePNGTextChunks(t *testing.T) {
	raw, errs := ParsePNG(buildPNG(
		pngChunk("tEXt", append([]byte("Software\x00"), []byte("NovelAI")...)),
		pngChunk("zTXt", buildZTXt("Comment", "compressed note")),
		pngChunk("iTXt", buildITXt("Description", "sunset over water")),
	))

	if len(errs) != 0 {
		t.Fatalf("ParsePNG returned errors: %v", errs)
	}
	if raw["Software"] != "NovelAI" {
		t.Fatalf("Software = %q", raw["Software"])
	}
	if raw["Comment"] != "compressed note" {
		t.Fatalf("Comment = %q", raw["Comment"])
	}
	if raw["Description"] != "sunset over water" {
		t.Fatalf("Description = %q", raw["Description"])
	}
}

func TestParsePNGReportsCorruptChunksAndContinues(t *testing.T) {
	raw, errs := ParsePNG(buildPNG(
		pngChunk("tEXt", []byte("ok\x00yes")),
		corruptPNGChunk("tEXt", []byte("bad\x00no")),
		pngChunk("tEXt", []byte("after\x00still read")),
	))

	if len(errs) == 0 {
		t.Fatal("expected a corrupt chunk error")
	}
	if raw["ok"] != "yes" || raw["after"] != "still read" {
		t.Fatalf("expected valid chunks to be retained, got %#v", raw)
	}
}

func TestParseWebPMetadataChunks(t *testing.T) {
	raw, errs := ParseWebP(buildWebP(
		webpChunk("EXIF", []byte("exif bytes")),
		webpChunk("XMP ", []byte("<x:xmpmeta>data</x:xmpmeta>")),
		webpChunk("JSON", []byte(`{"prompt":"city lights","seed":42}`)),
		webpChunk("ABCD", []byte{0xff, 0x00, 0x01}),
	))

	if len(errs) != 0 {
		t.Fatalf("ParseWebP returned errors: %v", errs)
	}
	if raw["EXIF"] != "exif bytes" {
		t.Fatalf("EXIF = %q", raw["EXIF"])
	}
	if raw["XMP"] != "<x:xmpmeta>data</x:xmpmeta>" {
		t.Fatalf("XMP = %q", raw["XMP"])
	}
	if raw["JSON.prompt"] != "city lights" || raw["JSON.seed"] != "42" {
		t.Fatalf("JSON fields not flattened: %#v", raw)
	}
	if _, ok := raw["ABCD"]; ok {
		t.Fatalf("non-text unknown chunk should be skipped: %#v", raw)
	}
}

func TestExtractWebPJSONChunkUsesCanonicalPromptKeys(t *testing.T) {
	raw, errs := ParseWebP(buildWebP(
		webpChunk("JSON", []byte(`{"prompt":"city lights, rain","uc":"blur","seed":42}`)),
	))
	if len(errs) != 0 {
		t.Fatalf("ParseWebP returned errors: %v", errs)
	}

	got := Extract(raw)

	if got.Source != SourceNAI {
		t.Fatalf("Source = %q, raw = %#v", got.Source, raw)
	}
	if got.PositivePrompt != "city lights, rain" || got.NegativePrompt != "blur" {
		t.Fatalf("prompts = %#v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0].Name != "city lights" || got.Tags[1].Name != "rain" {
		t.Fatalf("tags = %#v", got.Tags)
	}
}

func TestExtractNovelAIFromRawMetadata(t *testing.T) {
	raw := RawMetadata{
		"Software":    "NovelAI",
		"Description": "masterpiece, blue sky",
		"Comment":     `{"uc":"low quality, blurry","seed":1234,"sampler":"k_euler","scale":7.5,"steps":28,"prompt":"json prompt"}`,
	}

	got := Extract(raw)

	if got.Source != SourceNAI {
		t.Fatalf("Source = %q", got.Source)
	}
	if got.PositivePrompt != "masterpiece, blue sky" {
		t.Fatalf("PositivePrompt = %q", got.PositivePrompt)
	}
	if got.NegativePrompt != "low quality, blurry" {
		t.Fatalf("NegativePrompt = %q", got.NegativePrompt)
	}
	if got.Settings["seed"] != "1234" || got.Settings["sampler"] != "k_euler" || got.Settings["scale"] != "7.5" {
		t.Fatalf("settings = %#v", got.Settings)
	}
	if len(got.Tags) != 2 || got.Tags[0].Name != "masterpiece" || got.Tags[1].Name != "blue sky" {
		t.Fatalf("tags = %#v", got.Tags)
	}
}

func TestExtractComfyUIWorkflowSummary(t *testing.T) {
	promptJSON := `{
		"3":{"class_type":"KSampler","inputs":{"sampler_name":"dpmpp_2m","scheduler":"karras","seed":99}},
		"4":{"class_type":"CheckpointLoaderSimple","inputs":{"ckpt_name":"anime.safetensors"}},
		"5":{"class_type":"CLIPTextEncode","inputs":{"text":"glowing forest"}}
	}`
	raw := RawMetadata{"prompt": promptJSON}

	got := Extract(raw)

	if got.Source != SourceComfyUI {
		t.Fatalf("Source = %q", got.Source)
	}
	if got.Workflow.NodeCount != 3 {
		t.Fatalf("NodeCount = %d", got.Workflow.NodeCount)
	}
	if !contains(got.Workflow.Checkpoints, "anime.safetensors") {
		t.Fatalf("Checkpoints = %#v", got.Workflow.Checkpoints)
	}
	if !contains(got.Workflow.Samplers, "dpmpp_2m") || !contains(got.Workflow.Schedulers, "karras") {
		t.Fatalf("workflow = %#v", got.Workflow)
	}
	if got.PositivePrompt != "glowing forest" {
		t.Fatalf("PositivePrompt = %q", got.PositivePrompt)
	}
}

func TestExtractComfyUINegativePromptFromKSamplerLinks(t *testing.T) {
	promptJSON := `{
		"1":{"class_type":"CLIPTextEncode","inputs":{"text":"glowing forest"}},
		"2":{"class_type":"CLIPTextEncode","inputs":{"text":"low quality"}},
		"3":{"class_type":"KSampler","inputs":{"positive":["1",0],"negative":["2",0]}}
	}`
	raw := RawMetadata{"prompt": promptJSON}

	got := Extract(raw)

	if got.PositivePrompt != "glowing forest" {
		t.Fatalf("PositivePrompt = %q", got.PositivePrompt)
	}
	if got.NegativePrompt != "low quality" {
		t.Fatalf("NegativePrompt = %q", got.NegativePrompt)
	}
}

func TestImageDimensionsReadsVP8AndVP8LWebP(t *testing.T) {
	vp8 := buildWebP(webpChunk("VP8 ", buildVP8FrameHeader(320, 240)))
	width, height := imageDimensions("webp", vp8)
	if width != 320 || height != 240 {
		t.Fatalf("VP8 dimensions = %dx%d", width, height)
	}

	vp8l := buildWebP(webpChunk("VP8L", buildVP8LHeader(123, 45)))
	width, height = imageDimensions("webp", vp8l)
	if width != 123 || height != 45 {
		t.Fatalf("VP8L dimensions = %dx%d", width, height)
	}
}

func TestExtractGenericAndUnknown(t *testing.T) {
	generic := Extract(RawMetadata{"Description": "one, two", "seed": "10"})
	if generic.Source != SourceGeneric {
		t.Fatalf("generic Source = %q", generic.Source)
	}
	if generic.PositivePrompt != "one, two" || generic.Settings["seed"] != "10" {
		t.Fatalf("generic metadata = %#v", generic)
	}

	unknown := Extract(RawMetadata{})
	if unknown.Source != SourceUnknown {
		t.Fatalf("unknown Source = %q", unknown.Source)
	}
}

func buildPNG(chunks ...[]byte) []byte {
	out := append([]byte{}, pngSignature[:]...)
	for _, chunk := range chunks {
		out = append(out, chunk...)
	}
	out = append(out, pngChunk("IEND", nil)...)
	return out
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

func corruptPNGChunk(kind string, data []byte) []byte {
	chunk := pngChunk(kind, data)
	chunk[len(chunk)-1] ^= 0xff
	return chunk
}

func buildZTXt(key, text string) []byte {
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	_, _ = zw.Write([]byte(text))
	_ = zw.Close()
	return append(append([]byte(key+"\x00"), 0), compressed.Bytes()...)
}

func buildITXt(key, text string) []byte {
	return []byte(key + "\x00\x00\x00\x00\x00" + text)
}

func buildWebP(chunks ...[]byte) []byte {
	var payload bytes.Buffer
	for _, chunk := range chunks {
		payload.Write(chunk)
	}
	var out bytes.Buffer
	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(payload.Len()+4))
	out.WriteString("WEBP")
	out.Write(payload.Bytes())
	return out.Bytes()
}

func webpChunk(kind string, data []byte) []byte {
	var out bytes.Buffer
	out.WriteString(kind)
	_ = binary.Write(&out, binary.LittleEndian, uint32(len(data)))
	out.Write(data)
	if len(data)%2 == 1 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

func buildVP8FrameHeader(width, height int) []byte {
	header := []byte{0x9d, 0x01, 0x2a, byte(width), byte(width >> 8), byte(height), byte(height >> 8)}
	return append([]byte{0x00, 0x00, 0x00}, header...)
}

func buildVP8LHeader(width, height int) []byte {
	w := uint32(width - 1)
	h := uint32(height - 1)
	bits := w | (h << 14)
	return []byte{0x2f, byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func TestWebPJSONFixtureIsValid(t *testing.T) {
	var v map[string]any
	if err := json.Unmarshal([]byte(`{"prompt":"city lights","seed":42}`), &v); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(v["prompt"].(string)) == "" {
		t.Fatal("fixture prompt is empty")
	}
}
