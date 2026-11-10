package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// decodedBody returns the body as stored for the UI: decoded per the
// Content-Encoding header (gzip, deflate, br, zstd; several in the order they were
// applied), so a gzip answer is shown as text. The client still gets the
// original bytes. The output is capped at maxBodySize+1 bytes, and a body cut
// at the capture limit decodes as far as it goes. An unknown encoding or one
// that fails at once leaves the body as it came.
func decodedBody(body []byte, headers http.Header) []byte {
	// Codings may come on one line or on several header lines.
	enc := strings.Join(headers.Values("Content-Encoding"), ",")
	if enc == "" || len(body) == 0 {
		return body
	}
	codings := strings.Split(enc, ",")
	out := body
	for i := len(codings) - 1; i >= 0; i-- {
		d, ok := decode(out, strings.ToLower(strings.TrimSpace(codings[i])))
		if !ok {
			return body
		}
		out = d
	}
	return out
}

func decode(data []byte, coding string) ([]byte, bool) {
	var r io.Reader
	switch coding {
	case "identity", "":
		return data, true
	case "gzip", "x-gzip":
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		r = zr
	case "deflate":
		// RFC 9110 deflate is zlib-wrapped; some servers send raw deflate.
		if zr, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
			r = zr
		} else {
			r = flate.NewReader(bytes.NewReader(data))
		}
	case "br":
		r = brotli.NewReader(bytes.NewReader(data))
	case "zstd":
		zr, err := zstd.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		defer zr.Close()
		r = zr
	default:
		return nil, false
	}
	out, err := io.ReadAll(io.LimitReader(r, maxBodySize+1))
	if len(out) == 0 && err != nil {
		return nil, false
	}
	return out, true
}
