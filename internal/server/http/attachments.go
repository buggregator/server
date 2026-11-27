package http

import (
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/buggregator/server/v2/internal/storage"
)

// RegisterAttachmentAPI registers attachment list/download/preview endpoints.
//
// Routes are namespaced under a literal `attachments` segment rather than
// `/api/{module}/{eventUuid}/attachments/...`. The previous layout used a
// wildcard at position 2, which collides with module routes like
// `/api/smtp/message/{uuid}/raw` (both 4-segment patterns, neither more
// specific) and made the server panic at startup.
//
// The list goes through protect (the API auth): the UI fetches it with the
// token. Download and preview stay open: the UI opens them as plain links,
// which carry no X-Auth-Token, and each needs both the event and attachment
// UUIDs, which only the protected API hands out.
func RegisterAttachmentAPI(mux *http.ServeMux, db *sql.DB, store *storage.AttachmentStore, protect func(http.Handler) http.Handler) {
	// SMTP attachments.
	registerAttachmentsForModule(mux, db, store, protect, "/api/smtp/attachments", "smtp_attachments")

	// HTTP Dump attachments (support both /http-dump/ and /http-dumps/ paths
	// for backwards compatibility with existing clients).
	registerAttachmentsForModule(mux, db, store, protect, "/api/http-dump/attachments", "http_dump_attachments")
	registerAttachmentsForModule(mux, db, store, protect, "/api/http-dumps/attachments", "http_dump_attachments")
}

func registerAttachmentsForModule(mux *http.ServeMux, db *sql.DB, store *storage.AttachmentStore, protect func(http.Handler) http.Handler, prefix, table string) {
	mux.Handle("GET "+prefix+"/{uuid}", protect(listAttachments(db, table)))
	mux.HandleFunc("GET "+prefix+"/{eventUuid}/{uuid}", downloadAttachment(db, store, table))
	mux.HandleFunc("GET "+prefix+"/{eventUuid}/preview/{uuid}", previewAttachment(db, store, table))
}

func listAttachments(db *sql.DB, table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventUUID := r.PathValue("uuid")

		rows, err := db.QueryContext(r.Context(),
			"SELECT uuid, name, path, size, mime FROM "+table+" WHERE event_uuid = ?", eventUUID)
		if err != nil {
			writeError(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		var attachments []map[string]any
		for rows.Next() {
			var uuid, name, path, mime string
			var size int
			rows.Scan(&uuid, &name, &path, &size, &mime)
			attachments = append(attachments, map[string]any{
				"uuid": uuid, "name": name, "path": path, "size": size, "mime": mime,
			})
		}
		if attachments == nil {
			attachments = []map[string]any{}
		}

		writeJSON(w, map[string]any{"data": attachments, "meta": map[string]any{}})
	}
}

func downloadAttachment(db *sql.DB, store *storage.AttachmentStore, table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventUUID := r.PathValue("eventUuid")
		uuid := r.PathValue("uuid")

		var name, path, mime string
		var size int
		var storedEventUUID string
		err := db.QueryRowContext(r.Context(),
			"SELECT event_uuid, name, path, size, mime FROM "+table+" WHERE uuid = ?", uuid,
		).Scan(&storedEventUUID, &name, &path, &size, &mime)
		if err != nil {
			http.Error(w, "attachment not found", 404)
			return
		}

		if storedEventUUID != eventUUID {
			http.Error(w, "forbidden", 403)
			return
		}

		data, err := store.Get(path)
		if err != nil {
			http.Error(w, "file not found", 404)
			return
		}

		setAttachmentSafetyHeaders(w, "same-origin")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", contentDisposition("attachment", name))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}

func previewAttachment(db *sql.DB, store *storage.AttachmentStore, table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventUUID := r.PathValue("eventUuid")
		uuid := r.PathValue("uuid")

		var name, path, mimeType string
		var storedEventUUID string
		err := db.QueryRowContext(r.Context(),
			"SELECT event_uuid, name, path, mime FROM "+table+" WHERE uuid = ?", uuid,
		).Scan(&storedEventUUID, &name, &path, &mimeType)
		if err != nil {
			http.Error(w, "attachment not found", 404)
			return
		}

		if storedEventUUID != eventUUID {
			http.Error(w, "forbidden", 403)
			return
		}

		data, err := store.Get(path)
		if err != nil {
			http.Error(w, "file not found", 404)
			return
		}

		// cross-origin: the UI shows mail HTML in a sandboxed srcdoc iframe
		// (opaque origin), and its cid: images load from here.
		setAttachmentSafetyHeaders(w, "cross-origin")
		if ct, ok := inlinePreviewType(mimeType); ok {
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Content-Disposition", contentDisposition("inline", name))
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", contentDisposition("attachment", name))
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}

// attachmentCSP keeps any attachment body away from the UI origin: the
// sandbox directive gives the document an opaque origin, and nothing but
// same-origin images and media may load.
const attachmentCSP = "default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'; sandbox"

func setAttachmentSafetyHeaders(w http.ResponseWriter, corp string) {
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", attachmentCSP)
	h.Set("Cross-Origin-Resource-Policy", corp)
}

// inlineImageTypes are raster image types a browser renders without running
// code. SVG is left out on purpose: it can carry script.
var inlineImageTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"image/webp":               true,
	"image/avif":               true,
	"image/bmp":                true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
}

// inlinePreviewType returns the Content-Type to serve a stored attachment
// inline with, and false when the type is not safe to show inline. The
// stored MIME comes from the sender, so only an allowlist passes.
func inlinePreviewType(stored string) (string, bool) {
	mediaType, _, err := mime.ParseMediaType(stored)
	if err != nil {
		return "", false
	}
	mediaType = strings.ToLower(mediaType)
	switch {
	case inlineImageTypes[mediaType], mediaType == "application/pdf":
		return mediaType, true
	case mediaType == "text/plain":
		return "text/plain; charset=utf-8", true
	case strings.HasPrefix(mediaType, "audio/"), strings.HasPrefix(mediaType, "video/"):
		return mediaType, true
	}
	return "", false
}

// contentDisposition builds a Content-Disposition value whose filename cannot
// break out of the header: the plain filename is an ASCII fallback with
// quotes, backslashes and control characters replaced, and a non-ASCII name
// also goes as RFC 5987 filename*.
func contentDisposition(kind, name string) string {
	var fallback strings.Builder
	ascii := true
	for _, r := range name {
		switch {
		case r > 0x7e:
			ascii = false
			fallback.WriteByte('_')
		case r < 0x20, r == 0x7f, r == '"', r == '\\':
			fallback.WriteByte('_')
		default:
			fallback.WriteRune(r)
		}
	}
	v := kind + "; filename=\"" + fallback.String() + "\""
	if !ascii {
		v += "; filename*=UTF-8''" + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
	}
	return v
}
