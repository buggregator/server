package sentry

import (
	"bytes"
	"compress/gzip"
	"errors"
	"runtime"
	"strings"
	"testing"
)

// allocated reports how many bytes f allocated.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func TestDecompress_Bombs(t *testing.T) {
	t.Run("zstd frame declaring a 512 MB window", func(t *testing.T) {
		bomb := []byte{0x28, 0xB5, 0x2F, 0xFD, 0x00, 0x98, 0x09, 0x00, 0x00, 0x61}
		var got []byte
		grew := allocated(func() { got = decompress(bomb, "zstd") })
		if grew > 64<<20 {
			t.Errorf("allocated %d MB for a 10-byte body", grew>>20)
		}
		if !bytes.Equal(got, bomb) {
			t.Errorf("body changed: %x", got)
		}
	})

	t.Run("gzip expanding to 256 MB", func(t *testing.T) {
		var buf bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		chunk := make([]byte, 1<<20)
		for i := 0; i < 256; i++ {
			zw.Write(chunk)
		}
		zw.Close()
		bomb := buf.Bytes()

		var got []byte
		grew := allocated(func() { got = decompress(bomb, "gzip") })
		// Capped: about two growing copies of the 64 MB limit. Uncapped, the
		// 256 MB body alone is more than limit allows.
		limit := uint64(200 << 20)
		if raceEnabled {
			limit *= 2
		}
		if grew > limit {
			t.Errorf("allocated %d MB for a %d KB body", grew>>20, len(bomb)>>10)
		}
		if !bytes.Equal(got, bomb) {
			t.Errorf("an over-limit body must stay as it came")
		}
	})
}

func TestReadCapped(t *testing.T) {
	if _, err := readCapped(strings.NewReader(strings.Repeat("a", maxPayloadSize))); err != nil {
		t.Errorf("exactly the limit: %v", err)
	}
	_, err := readCapped(strings.NewReader(strings.Repeat("a", maxPayloadSize+1)))
	if !errors.Is(err, errPayloadTooLarge) {
		t.Errorf("over the limit: err = %v, want errPayloadTooLarge", err)
	}
}
