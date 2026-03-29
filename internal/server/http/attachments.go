package http

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/buggregator/go-buggregator/internal/storage"
)

// RegisterAttachmentAPI registers attachment list/download/preview endpoints.
func RegisterAttachmentAPI(mux *http.ServeMux, db *sql.DB, store *storage.AttachmentStore) {
	// SMTP attachments.
	mux.HandleFunc("GET /api/smtp/{uuid}/attachments", listAttachments(db, "smtp_attachments"))
	mux.HandleFunc("GET /api/smtp/{eventUuid}/attachments/{uuid}", downloadAttachment(db, store, "smtp_attachments"))
	mux.HandleFunc("GET /api/smtp/{eventUuid}/attachments/preview/{uuid}", previewAttachment(db, store, "smtp_attachments"))

	// HTTP Dump attachments (support both /http-dump/ and /http-dumps/ paths).
	for _, prefix := range []string{"/api/http-dump/", "/api/http-dumps/"} {
		mux.HandleFunc("GET "+prefix+"{uuid}/attachments", listAttachments(db, "http_dump_attachments"))
		mux.HandleFunc("GET "+prefix+"{eventUuid}/attachments/{uuid}", downloadAttachment(db, store, "http_dump_attachments"))
		mux.HandleFunc("GET "+prefix+"{eventUuid}/attachments/preview/{uuid}", previewAttachment(db, store, "http_dump_attachments"))
	}
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

