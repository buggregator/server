package selfupdate

import (
	"encoding/json"
	"errors"
	"math"
	"mime"
	"net/http"
)

// Register adds GET /api/update, POST /api/update/check and POST
// /api/update behind the API's auth middleware.
func (u *Updater) Register(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	mux.Handle("GET /api/update", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, u.Info())
	})))

	mux.Handle("POST /api/update/check", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeErr(w, http.StatusForbidden, "cross-site requests cannot check for updates")
			return
		}
		wait, err := u.CheckNow(r.Context())
		if wait > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message":     "checked less than a minute ago",
				"code":        http.StatusTooManyRequests,
				"retry_after": int(math.Ceil(wait.Seconds())),
			})
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadGateway, "update check failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, u.Info())
	})))

	mux.Handle("POST /api/update", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeErr(w, http.StatusForbidden, "cross-site requests cannot start an update")
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			writeErr(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}
		var body struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || body.Version == "" {
			writeErr(w, http.StatusBadRequest, `body must be {"version": "X.Y.Z"}`)
			return
		}
		switch err := u.Start(body.Version); {
		case err == nil:
			writeJSON(w, http.StatusAccepted, u.Info())
		case errors.Is(err, ErrNotAllowed):
			writeErr(w, http.StatusForbidden, u.mode.Reason)
		case errors.Is(err, ErrBusy):
			writeErr(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrNotLatest):
			writeErr(w, http.StatusBadRequest, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
	})))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"message": msg, "code": code})
}
