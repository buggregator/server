package ray

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// Pause states of a ray()->pause() lock, as the UI sees them in the
// create_lock payload's content.state.
const (
	LockActive    = "active"
	LockContinued = "continued"
	LockStopped   = "stopped"
)

// lockTTL is how long a lock is remembered: a paused script polls every
// second, so a lock nobody released for a day belongs to a script long gone.
const lockTTL = 24 * time.Hour

type lock struct {
	uuid    string // event holding the create_lock payload
	state   string
	created time.Time
}

// locks keeps the pauses of ray()->pause() in memory. The PHP client sends
// create_lock, then polls GET /locks/{name} every second until the lock is
// not active; stop_execution makes it throw StopExecutionRequested. A server
// restart forgets the locks, and a paused script then goes on.
type locks struct {
	mu  sync.Mutex
	m   map[string]*lock
	now func() time.Time
}

func newLocks() *locks {
	return &locks{m: map[string]*lock{}, now: time.Now}
}

// create registers an active lock held by event uuid.
func (l *locks) create(name, uuid string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for n, lk := range l.m {
		if now.Sub(lk.created) > lockTTL {
			delete(l.m, n)
		}
	}
	l.m[name] = &lock{uuid: uuid, state: LockActive, created: now}
}

// get returns the lock's state and event uuid; ok is false for an unknown lock.
func (l *locks) get(name string) (state, uuid string, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lk, ok := l.m[name]
	if !ok {
		return "", "", false
	}
	return lk.state, lk.uuid, true
}

// release moves an active lock to continued or stopped. changed is false
// when the lock is unknown or already released (a second click, a second tab).
func (l *locks) release(name string, stop bool) (uuid, state string, changed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lk, ok := l.m[name]
	if !ok {
		return "", "", false
	}
	if lk.state != LockActive {
		return lk.uuid, lk.state, false
	}
	lk.state = LockContinued
	if stop {
		lk.state = LockStopped
	}
	return lk.uuid, lk.state, true
}

// lockNames returns the names of the create_lock payloads of a Ray request.
func lockNames(body []byte) []string {
	var req struct {
		Payloads []struct {
			Type    string `json:"type"`
			Content struct {
				Name string `json:"name"`
			} `json:"content"`
		} `json:"payloads"`
	}
	if json.Unmarshal(body, &req) != nil {
		return nil
	}
	var names []string
	for _, p := range req.Payloads {
		if p.Type == "create_lock" && p.Content.Name != "" {
			names = append(names, p.Content.Name)
		}
	}
	return names
}

// setLockState writes state into the create_lock payload named name, so a
// reloaded page shows the pause as released.
func setLockState(name, state string) func(json.RawMessage) (json.RawMessage, error) {
	return func(stored json.RawMessage) (json.RawMessage, error) {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(stored, &doc); err != nil {
			return nil, err
		}
		var payloads []map[string]json.RawMessage
		if err := json.Unmarshal(doc["payloads"], &payloads); err != nil {
			return nil, err
		}
		for _, p := range payloads {
			var typ string
			_ = json.Unmarshal(p["type"], &typ)
			if typ != "create_lock" {
				continue
			}
			var content map[string]any
			if err := json.Unmarshal(p["content"], &content); err != nil {
				continue
			}
			if content["name"] != name {
				continue
			}
			content["state"] = state
			raw, err := json.Marshal(content)
			if err != nil {
				return nil, err
			}
			p["content"] = raw
		}
		raw, err := json.Marshal(payloads)
		if err != nil {
			return nil, err
		}
		doc["payloads"] = raw
		return json.Marshal(doc)
	}
}

// handleLockStatus answers the PHP client's poll, GET /locks/{name}:
// {"active": bool, "stop_execution": bool}. An unknown lock is not active.
func (m *Module) handleLockStatus(w http.ResponseWriter, r *http.Request) {
	state, _, _ := m.locks.get(r.PathValue("name"))
	writeJSON(w, map[string]bool{
		"active":         state == LockActive,
		"stop_execution": state == LockStopped,
	})
}

// handleLockRelease is the UI's Continue / Stop execution:
// POST /api/ray/locks/{name}, body {"stop_execution": true} to stop.
// It answers {"status": true, "state": "..."}; 404 for an unknown lock.
func (m *Module) handleLockRelease(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StopExecution bool `json:"stop_execution"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body)

	name := r.PathValue("name")
	uuid, state, changed := m.locks.release(name, body.StopExecution)
	if uuid == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": false, "error": "lock not found"})
		return
	}
	if changed {
		if err := m.saveLockState(r.Context(), uuid, name, state); err != nil {
			writeJSON(w, map[string]any{"status": true, "state": state, "warning": err.Error()})
			return
		}
	}
	writeJSON(w, map[string]any{"status": true, "state": state})
}

var errNoEventService = errors.New("no event service")

// saveLockState stores the new state in the event and re-broadcasts it, so
// every open tab disables the buttons. The pause itself is already released.
func (m *Module) saveLockState(ctx context.Context, uuid, name, state string) error {
	store := m.store
	if m.events == nil || store == nil {
		return errNoEventService
	}
	stored, err := store.FindByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	if stored == nil {
		return nil // the event was deleted: nothing to show the state on
	}
	return m.events.HandleIncoming(ctx, &event.Incoming{
		UUID:    uuid,
		Type:    "ray",
		Project: stored.Project,
		Payload: stored.Payload,
		Merge:   setLockState(name, state),
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
