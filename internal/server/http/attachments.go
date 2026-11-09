package http

import (
	"database/sql"
	"fmt"
	"net/http"

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

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}

func previewAttachment(db *sql.DB, store *storage.AttachmentStore, table string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventUUID := r.PathValue("eventUuid")
		uuid := r.PathValue("uuid")

		var name, path, mime string
		var storedEventUUID string
		err := db.QueryRowContext(r.Context(),
			"SELECT event_uuid, name, path, mime FROM "+table+" WHERE uuid = ?", uuid,
		).Scan(&storedEventUUID, &name, &path, &mime)
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

		w.Header().Set("Content-Type", mime)
		w.Write(data)
	}
}

