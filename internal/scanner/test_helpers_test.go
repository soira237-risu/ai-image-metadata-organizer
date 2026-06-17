package scanner

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
)

func writePNGTextFixture(path string, chunks map[string]string) error {
	var out bytes.Buffer
	out.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	out.Write(pngChunk("IHDR", buildIHDR(1, 1)))
	for key, value := range chunks {
		out.Write(pngChunk("tEXt", append(append([]byte{}, []byte(key)...), append([]byte{0}, []byte(value)...)...)))
	}
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

func writeWebPJSONFixture(path string, jsonText string) error {
	var payload bytes.Buffer
	payload.Write(webpChunk("JSON", []byte(jsonText)))
	var out bytes.Buffer
	out.WriteString("RIFF")
	_ = binary.Write(&out, binary.LittleEndian, uint32(payload.Len()+4))
	out.WriteString("WEBP")
	out.Write(payload.Bytes())
	return os.WriteFile(path, out.Bytes(), 0644)
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
