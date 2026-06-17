package metadata

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

var pngSignature = [8]byte{137, 80, 78, 71, 13, 10, 26, 10}

func ParsePNG(data []byte) (RawMetadata, []error) {
	raw := RawMetadata{}
	var errs []error
	if len(data) < len(pngSignature) || !bytes.Equal(data[:len(pngSignature)], pngSignature[:]) {
		return raw, []error{errors.New("invalid PNG signature")}
	}

	offset := len(pngSignature)
	for offset+12 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		kind := string(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length+4 > len(data) {
			errs = append(errs, fmt.Errorf("corrupt PNG chunk %q: declared length exceeds file", kind))
			break
		}
		chunkData := data[offset : offset+length]
		offset += length
		gotCRC := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		wantCRC := crc32.ChecksumIEEE(append([]byte(kind), chunkData...))
		if gotCRC != wantCRC {
			errs = append(errs, fmt.Errorf("corrupt PNG chunk %q: CRC mismatch", kind))
			if kind == "IEND" {
				break
			}
			continue
		}

		switch kind {
		case "tEXt":
			parsePNGText(raw, chunkData)
		case "zTXt":
			if err := parsePNGZText(raw, chunkData); err != nil {
				errs = append(errs, err)
			}
		case "iTXt":
			if err := parsePNGInternationalText(raw, chunkData); err != nil {
				errs = append(errs, err)
			}
		case "IEND":
			return raw, errs
		}
	}
	if offset < len(data) {
		errs = append(errs, errors.New("trailing PNG data after incomplete chunk"))
	}
	return raw, errs
}

func parsePNGText(raw RawMetadata, data []byte) {
	parts := bytes.SplitN(data, []byte{0}, 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return
	}
	raw[string(parts[0])] = string(parts[1])
}

func parsePNGZText(raw RawMetadata, data []byte) error {
	parts := bytes.SplitN(data, []byte{0}, 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return errors.New("invalid zTXt chunk")
	}
	if len(parts[1]) < 1 || parts[1][0] != 0 {
		return fmt.Errorf("unsupported zTXt compression method for %q", string(parts[0]))
	}
	zr, err := zlib.NewReader(bytes.NewReader(parts[1][1:]))
	if err != nil {
		return fmt.Errorf("invalid zTXt data for %q: %w", string(parts[0]), err)
	}
	defer zr.Close()
	text, err := io.ReadAll(zr)
	if err != nil {
		return fmt.Errorf("invalid zTXt data for %q: %w", string(parts[0]), err)
	}
	raw[string(parts[0])] = string(text)
	return nil
}

func parsePNGInternationalText(raw RawMetadata, data []byte) error {
	first := bytes.IndexByte(data, 0)
	if first <= 0 {
		return errors.New("invalid iTXt chunk")
	}
	key := string(data[:first])
	rest := data[first+1:]
	if len(rest) < 2 {
		return fmt.Errorf("invalid iTXt chunk for %q", key)
	}
	compressed := rest[0] == 1
	method := rest[1]
	rest = rest[2:]
	langEnd := bytes.IndexByte(rest, 0)
	if langEnd < 0 {
		return fmt.Errorf("invalid iTXt language tag for %q", key)
	}
	rest = rest[langEnd+1:]
	translatedEnd := bytes.IndexByte(rest, 0)
	if translatedEnd < 0 {
		return fmt.Errorf("invalid iTXt translated keyword for %q", key)
	}
	textBytes := rest[translatedEnd+1:]
	if compressed {
		if method != 0 {
			return fmt.Errorf("unsupported iTXt compression method for %q", key)
		}
		zr, err := zlib.NewReader(bytes.NewReader(textBytes))
		if err != nil {
			return fmt.Errorf("invalid iTXt data for %q: %w", key, err)
		}
		defer zr.Close()
		readBytes, readErr := io.ReadAll(zr)
		if readErr != nil {
			return fmt.Errorf("invalid iTXt data for %q: %w", key, readErr)
		}
		textBytes = readBytes
	}
	raw[key] = string(textBytes)
	return nil
}
