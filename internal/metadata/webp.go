package metadata

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

func ParseWebP(data []byte) (RawMetadata, []error) {
	raw := RawMetadata{}
	var errs []error
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return raw, []error{errors.New("invalid WebP RIFF header")}
	}

	riffSize := int(binary.LittleEndian.Uint32(data[4:8]))
	limit := len(data)
	if riffSize+8 < limit {
		limit = riffSize + 8
	}
	offset := 12
	for offset+8 <= limit {
		kind := string(data[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if size < 0 || offset+size > limit {
			errs = append(errs, fmt.Errorf("corrupt WebP chunk %q: declared length exceeds file", kind))
			break
		}
		chunkData := data[offset : offset+size]
		offset += size
		if size%2 == 1 && offset < limit {
			offset++
		}

		key := strings.TrimSpace(kind)
		switch kind {
		case "EXIF":
			raw["EXIF"] = string(chunkData)
		case "XMP ":
			raw["XMP"] = string(chunkData)
		default:
			if !utf8.Valid(chunkData) {
				continue
			}
			text := strings.TrimSpace(string(chunkData))
			if text == "" {
				continue
			}
			if flattenJSON(raw, key, text) {
				continue
			}
			if isLikelyTextChunk(kind, text) {
				raw[key] = text
			}
		}
	}
	if offset < limit {
		errs = append(errs, errors.New("trailing WebP data after incomplete chunk"))
	}
	return raw, errs
}

func isLikelyTextChunk(kind, text string) bool {
	upper := strings.ToUpper(strings.TrimSpace(kind))
	if strings.Contains(upper, "TXT") || strings.Contains(upper, "JSON") || strings.Contains(upper, "META") {
		return true
	}
	return strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") || strings.Contains(text, "=")
}

func flattenJSON(raw RawMetadata, prefix, text string) bool {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return false
	}
	flattenJSONValue(raw, prefix, value)
	return true
}

func flattenJSONValue(raw RawMetadata, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenJSONValue(raw, next, child)
			if isCanonicalJSONKey(key) {
				flattenJSONValue(raw, key, child)
			}
		}
	case []any:
		encoded, err := json.Marshal(typed)
		if err == nil {
			raw[prefix] = string(encoded)
		}
	case string:
		raw[prefix] = typed
	case float64, bool, nil:
		encoded, err := json.Marshal(typed)
		if err == nil {
			raw[prefix] = string(encoded)
		}
	default:
		encoded, err := json.Marshal(typed)
		if err == nil {
			raw[prefix] = string(encoded)
		}
	}
}

func isCanonicalJSONKey(key string) bool {
	switch key {
	case "prompt", "workflow", "uc", "negative_prompt", "seed", "sampler", "sampler_name", "scale", "cfg_scale", "steps", "scheduler":
		return true
	default:
		return false
	}
}
