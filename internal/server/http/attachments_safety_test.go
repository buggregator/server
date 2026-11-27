package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serverhttp "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/storage"
)

func setupAttachmentMux(t *testing.T) (*http.ServeMux, func(uuid, name, mime string)) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	for _, table := range []string{"smtp_attachments", "http_dump_attachments"} {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + table + ` (
			uuid TEXT PRIMARY KEY, event_uuid TEXT NOT NULL, name TEXT NOT NULL, path TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0, mime TEXT NOT NULL DEFAULT '', content_id TEXT DEFAULT ''
		)`); err != nil {
			t.Fatal(err)
		}
	}

	store := storage.NewAttachmentStore("memory", "")
	mux := http.NewServeMux()
	serverhttp.RegisterAttachmentAPI(mux, db, store, func(next http.Handler) http.Handler { return next })

	add := func(uuid, name, mime string) {
		t.Helper()
		path := "ev/" + uuid
		if err := store.Store(path, []byte("<script>alert(1)</script>")); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"smtp_attachments", "http_dump_attachments"} {
			if _, err := db.Exec(`INSERT INTO `+table+` (uuid, event_uuid, name, path, mime) VALUES (?, 'ev', ?, ?, ?)`,
				uuid, name, path, mime); err != nil {
				t.Fatal(err)
			}
		}
	}
	return mux, add
}

func assertSafetyHeaders(t *testing.T, h http.Header, wantCORP string) {
	t.Helper()
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if csp := h.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") || !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q", csp)
	}
	if got := h.Get("Cross-Origin-Resource-Policy"); got != wantCORP {
		t.Errorf("Cross-Origin-Resource-Policy = %q", got)
	}
}

func TestAttachmentPreview_OnlySafeTypesInline(t *testing.T) {
	mux, add := setupAttachmentMux(t)

	cases := []struct {
		uuid, mime string
		wantType   string
		wantInline bool
	}{
		{"html", "text/html", "application/octet-stream", false},
		{"htmlcs", "TEXT/HTML; charset=utf-8", "application/octet-stream", false},
		{"svg", "image/svg+xml", "application/octet-stream", false},
		{"xml", "application/xml", "application/octet-stream", false},
		{"xhtml", "application/xhtml+xml", "application/octet-stream", false},
		{"js", "application/javascript", "application/octet-stream", false},
		{"unknown", "application/x-whatever", "application/octet-stream", false},
		{"empty", "", "application/octet-stream", false},
		{"garbage", ";;;", "application/octet-stream", false},
		{"png", "image/png", "image/png", true},
		{"jpegup", "IMAGE/JPEG", "image/jpeg", true},
		{"pdf", "application/pdf", "application/pdf", true},
		{"txt", "text/plain; charset=koi8-r", "text/plain; charset=utf-8", true},
		{"mp3", "audio/mpeg", "audio/mpeg", true},
		{"mp4", "video/mp4", "video/mp4", true},
	}
	for _, c := range cases {
		add(c.uuid, c.uuid+".bin", c.mime)
	}

	for _, prefix := range []string{"/api/smtp/attachments", "/api/http-dump/attachments", "/api/http-dumps/attachments"} {
		for _, c := range cases {
			t.Run(prefix+"/"+c.uuid, func(t *testing.T) {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest("GET", prefix+"/ev/preview/"+c.uuid, nil))
				if rec.Code != 200 {
					t.Fatalf("status %d", rec.Code)
				}
				// Preview must load from the UI's sandboxed (opaque-origin) mail iframe.
				assertSafetyHeaders(t, rec.Header(), "cross-origin")
				if got := rec.Header().Get("Content-Type"); got != c.wantType {
					t.Errorf("Content-Type = %q, want %q", got, c.wantType)
				}
				disp := rec.Header().Get("Content-Disposition")
				if c.wantInline && !strings.HasPrefix(disp, "inline;") {
					t.Errorf("Content-Disposition = %q, want inline", disp)
				}
				if !c.wantInline && !strings.HasPrefix(disp, "attachment;") {
					t.Errorf("Content-Disposition = %q, want attachment", disp)
				}
			})
		}
	}
}

func TestAttachmentDownload_SafeHeadersAndFilename(t *testing.T) {
	mux, add := setupAttachmentMux(t)
	add("quote", "a\"b\\c\r\nSet-Cookie: x=1.html", "text/html")
	add("utf", "отчёт \"1\".pdf", "application/pdf")
	add("plain", "report.pdf", "application/pdf")

	cases := []struct {
		uuid string
		want string
	}{
		{"quote", `attachment; filename="a_b_c__Set-Cookie: x=1.html"`},
		{"utf", `attachment; filename="_____ _1_.pdf"; filename*=UTF-8''%D0%BE%D1%82%D1%87%D1%91%D1%82%20%221%22.pdf`},
		{"plain", `attachment; filename="report.pdf"`},
	}
	for _, c := range cases {
		t.Run(c.uuid, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/smtp/attachments/ev/"+c.uuid, nil))
			if rec.Code != 200 {
				t.Fatalf("status %d", rec.Code)
			}
			assertSafetyHeaders(t, rec.Header(), "same-origin")
			if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := rec.Header().Get("Content-Disposition"); got != c.want {
				t.Errorf("Content-Disposition = %q, want %q", got, c.want)
			}
		})
	}

	// Preview uses the same filename escaping.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/smtp/attachments/ev/preview/utf", nil))
	if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, `inline; filename="_____ _1_.pdf"; filename*=UTF-8''`) {
		t.Errorf("preview Content-Disposition = %q", got)
	}
}
