package metadata

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ExtractFile(path string) (ImageMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImageMetadata{}, err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	var raw RawMetadata
	switch format {
	case "png":
		raw, _ = ParsePNG(data)
	case "webp":
		raw, _ = ParseWebP(data)
	default:
		return ImageMetadata{}, fmt.Errorf("unsupported image format %q", format)
	}
	width, height := imageDimensions(format, data)
	extracted := Extract(raw)
	record := Record{
		Source:          extracted.Source,
		Raw:             rawToAny(extracted.Raw),
		PositivePrompt:  extracted.PositivePrompt,
		NegativePrompt:  extracted.NegativePrompt,
		Settings:        stringMapToAny(extracted.Settings),
		WorkflowSummary: workflowToMap(extracted.Workflow),
	}
	tags := make([]ImageTag, 0, len(extracted.Tags))
	for _, tag := range extracted.Tags {
		kind := tag.Kind
		if kind == "" {
			kind = tag.Source
		}
		if kind == "" {
			kind = "prompt"
		}
		tags = append(tags, ImageTag{
			Value:      tag.Name,
			Normalized: normalizeTag(tag.Name),
			Source:     extracted.Source,
			Kind:       kind,
		})
	}
	return ImageMetadata{
		Format:   format,
		Width:    width,
		Height:   height,
		Metadata: []Record{record},
		Tags:     tags,
	}, nil
}

func imageDimensions(format string, data []byte) (int, int) {
	switch format {
	case "png":
		if len(data) >= 24 && string(data[:8]) == string(pngSignature[:]) {
			return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24]))
		}
	case "webp":
		return webpDimensions(data)
	}
	return 0, 0
}

func webpDimensions(data []byte) (int, int) {
	if len(data) < 20 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0
	}
	offset := 12
	for offset+8 <= len(data) {
		kind := string(data[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if size < 0 || offset+size > len(data) {
			return 0, 0
		}
		chunk := data[offset : offset+size]
		switch kind {
		case "VP8X":
			if len(chunk) >= 10 {
				width := 1 + int(chunk[4]) + int(chunk[5])<<8 + int(chunk[6])<<16
				height := 1 + int(chunk[7]) + int(chunk[8])<<8 + int(chunk[9])<<16
				return width, height
			}
		case "VP8 ":
			if len(chunk) >= 10 && chunk[3] == 0x9d && chunk[4] == 0x01 && chunk[5] == 0x2a {
				width := int(binary.LittleEndian.Uint16(chunk[6:8]) & 0x3fff)
				height := int(binary.LittleEndian.Uint16(chunk[8:10]) & 0x3fff)
				return width, height
			}
		case "VP8L":
			if len(chunk) >= 5 && chunk[0] == 0x2f {
				bits := uint32(chunk[1]) | uint32(chunk[2])<<8 | uint32(chunk[3])<<16 | uint32(chunk[4])<<24
				width := int(bits&0x3fff) + 1
				height := int((bits>>14)&0x3fff) + 1
				return width, height
			}
		}
		offset += size
		if size%2 == 1 {
			offset++
		}
	}
	return 0, 0
}

func rawToAny(raw RawMetadata) map[string]any {
	out := map[string]any{}
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func stringMapToAny(values map[string]string) map[string]any {
	out := map[string]any{}
	for key, value := range values {
		out[key] = value
	}
	return out
}

func workflowToMap(summary WorkflowSummary) map[string]any {
	return map[string]any{
		"node_count":  summary.NodeCount,
		"models":      summary.Models,
		"checkpoints": summary.Checkpoints,
		"samplers":    summary.Samplers,
		"schedulers":  summary.Schedulers,
	}
}

func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
