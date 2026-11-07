package vardumper

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func TestWriteParser(t *testing.T) {
	raw := []byte("\x7fELF-or-Mach-O parser bytes\x00\x01\x02")
	var gz bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	zw.Write(raw)
	zw.Close()

	tests := []struct {
		name string
		in   []byte
	}{
		{"plain bytes are written as they are", raw},
		{"gzipped bytes are unpacked", gz.Bytes()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := writeParser(&out, tt.in); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out.Bytes(), raw) {
				t.Fatalf("got %q, want %q", out.Bytes(), raw)
			}
		})
	}

	t.Run("broken gzip fails", func(t *testing.T) {
		var out bytes.Buffer
		if err := writeParser(&out, gz.Bytes()[:len(gz.Bytes())/2]); err == nil {
			t.Fatal("want an error for a truncated gzip stream")
		}
	})
}
