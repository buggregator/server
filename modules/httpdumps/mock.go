package httpdumps

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// Mock rules: "method + path → status, headers, body, delay" (#332).
// Requests under MockPrefix are answered by the first enabled matching rule and
// are always captured as HTTP dumps.

const (
	MockPrefix   = "/mock"
	MaxMockBody  = 1 << 20 // 1 MB
	MaxMockDelay = 30000   // ms
)

var mockMethods = map[string]bool{
	"ANY": true, http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
	http.MethodPatch: true, http.MethodDelete: true, http.MethodHead: true, http.MethodOptions: true,
}

// MockRule is one mock response. Rules from buggregator.yaml have Source
// "config" and are read-only; rules made through the API have Source "api".
type MockRule struct {
	ID        string            `json:"id"`
	Project   string            `json:"project"` // empty = every project
	Name      string            `json:"name"`
	Method    string            `json:"method"` // GET, POST, … or ANY
	Path      string            `json:"path"`   // /users/me, /users/{id}, /files/*
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	DelayMs   int               `json:"delay_ms"`
	Enabled   bool              `json:"enabled"`
	Position  int               `json:"position"`
	Source    string            `json:"source"`
	ReadOnly  bool              `json:"read_only"`
	CreatedAt string            `json:"created_at,omitempty"`
	UpdatedAt string            `json:"updated_at,omitempty"`
}

// ValidationError lists invalid fields of a rule: field → message.
type ValidationError map[string]string

func (v ValidationError) Error() string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + v[k]
	}
	return "invalid mock rule: " + strings.Join(parts, "; ")
}

// Normalize fills defaults and validates the rule.
func (r *MockRule) Normalize() error {
	errs := ValidationError{}

	r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
	if r.Method == "" {
		r.Method = "ANY"
	}
	if !mockMethods[r.Method] {
		errs["method"] = "must be one of GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS or ANY"
	}

	r.Path = strings.TrimSpace(r.Path)
	if err := validateMockPath(r.Path); err != nil {
		errs["path"] = err.Error()
	}

	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	if r.Status < 100 || r.Status > 599 {
		errs["status"] = "must be between 100 and 599"
	}
	if r.DelayMs < 0 || r.DelayMs > MaxMockDelay {
		errs["delay_ms"] = fmt.Sprintf("must be between 0 and %d", MaxMockDelay)
	}
	if len(r.Body) > MaxMockBody {
		errs["body"] = "must be at most 1 MB"
	}
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	for k := range r.Headers {
		if strings.TrimSpace(k) == "" || strings.ContainsAny(k, " :\r\n") {
			errs["headers"] = fmt.Sprintf("invalid header name %q", k)
		}
	}
	r.Project = strings.TrimSpace(r.Project)
	r.Name = strings.TrimSpace(r.Name)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateMockPath(p string) error {
	if !strings.HasPrefix(p, "/") {
		return errors.New("must start with /")
	}
	segs := splitPath(p)
	for i, s := range segs {
		switch {
		case s == "*":
			if i != len(segs)-1 {
				return errors.New("* is allowed only as the last segment")
			}
		case strings.Contains(s, "*"):
			return errors.New("* must be a whole segment")
		case strings.HasPrefix(s, "{") || strings.HasSuffix(s, "}"):
			if len(s) < 3 || !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
				return fmt.Errorf("bad parameter segment %q, use {name}", s)
			}
		}
	}
	return nil
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// matchMockPath reports whether path matches pattern: exact segments,
// {param} for one segment, and a trailing * for the rest (zero or more segments).
// A trailing slash is ignored on both sides.
func matchMockPath(pattern, path string) bool {
	ps, xs := splitPath(pattern), splitPath(path)
	for i, p := range ps {
		if p == "*" {
			return true
		}
		if i >= len(xs) {
			return false
		}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			continue
		}
		if p != xs[i] {
			return false
		}
	}
	return len(ps) == len(xs)
}

func (r *MockRule) matches(method, path, project string) bool {
	if !r.Enabled {
		return false
	}
	if r.Project != "" && r.Project != project {
		return false
	}
	if r.Method != "ANY" && r.Method != method {
		return false
	}
	return matchMockPath(r.Path, path)
}

// mockStore keeps the rules: config rules first (read-only), then the stored
// ones by position. The stored rules are cached and reloaded after each write,
// so matching a request does no SQL.
type mockStore struct {
	db *sql.DB

	mu     sync.RWMutex
	config []MockRule
	stored []MockRule
}

var errMockNotFound = errors.New("mock rule not found")
var errMockReadOnly = errors.New("mock rule comes from the config file and is read-only")

func newMockStore(db *sql.DB) *mockStore {
	return &mockStore{db: db}
}

func (s *mockStore) setConfig(rules []MockRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = make([]MockRule, len(rules))
	for i, r := range rules {
		r.ID = fmt.Sprintf("config-%d", i+1)
		r.Source = "config"
		r.ReadOnly = true
		r.Position = i
		s.config[i] = r
	}
}

func (s *mockStore) reload() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query(`SELECT id, COALESCE(project, ''), name, method, path, status, headers, body,
		delay_ms, enabled, position, created_at, updated_at FROM mock_rules ORDER BY position, created_at`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var rules []MockRule
	for rows.Next() {
		var r MockRule
		var headers string
		var enabled int
		if err := rows.Scan(&r.ID, &r.Project, &r.Name, &r.Method, &r.Path, &r.Status, &headers, &r.Body,
			&r.DelayMs, &enabled, &r.Position, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return err
		}
		r.Enabled = enabled == 1
		r.Source = "api"
		json.Unmarshal([]byte(headers), &r.Headers)
		if r.Headers == nil {
			r.Headers = map[string]string{}
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.stored = rules
	s.mu.Unlock()
	return nil
}

// list returns every rule in matching order.
func (s *mockStore) list() []MockRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MockRule, 0, len(s.config)+len(s.stored))
	out = append(out, s.config...)
	return append(out, s.stored...)
}

func (s *mockStore) get(id string) (MockRule, error) {
	for _, r := range s.list() {
		if r.ID == id {
			return r, nil
		}
	}
	return MockRule{}, errMockNotFound
}

// match returns the first enabled rule for the request, or nil.
func (s *mockStore) match(method, path, project string) *MockRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, list := range [][]MockRule{s.config, s.stored} {
		for i := range list {
			if list[i].matches(method, path, project) {
				r := list[i]
				return &r
			}
		}
	}
	return nil
}

func (s *mockStore) create(r MockRule) (MockRule, error) {
	if err := r.Normalize(); err != nil {
		return MockRule{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	r.ID = event.GenerateUUID()
	r.Source = "api"
	r.ReadOnly = false
	r.CreatedAt, r.UpdatedAt = now, now

	var next int
	s.db.QueryRow(`SELECT COALESCE(MAX(position), -1) + 1 FROM mock_rules`).Scan(&next)
	r.Position = next

	headers, _ := json.Marshal(r.Headers)
	if _, err := s.db.Exec(`INSERT INTO mock_rules
		(id, project, name, method, path, status, headers, body, delay_ms, enabled, position, created_at, updated_at)
		VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Project, r.Name, r.Method, r.Path, r.Status, string(headers), r.Body, r.DelayMs, boolInt(r.Enabled),
		r.Position, r.CreatedAt, r.UpdatedAt); err != nil {
		return MockRule{}, err
	}
	return r, s.reload()
}

func (s *mockStore) update(id string, r MockRule) (MockRule, error) {
	old, err := s.get(id)
	if err != nil {
		return MockRule{}, err
	}
	if old.ReadOnly {
		return MockRule{}, errMockReadOnly
	}
	if err := r.Normalize(); err != nil {
		return MockRule{}, err
	}
	r.ID, r.Source, r.Position, r.CreatedAt = id, "api", old.Position, old.CreatedAt
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	headers, _ := json.Marshal(r.Headers)
	if _, err := s.db.Exec(`UPDATE mock_rules SET project = NULLIF(?, ''), name = ?, method = ?, path = ?, status = ?,
		headers = ?, body = ?, delay_ms = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		r.Project, r.Name, r.Method, r.Path, r.Status, string(headers), r.Body, r.DelayMs, boolInt(r.Enabled),
		r.UpdatedAt, id); err != nil {
		return MockRule{}, err
	}
	return r, s.reload()
}

func (s *mockStore) delete(id string) error {
	old, err := s.get(id)
	if err != nil {
		return err
	}
	if old.ReadOnly {
		return errMockReadOnly
	}
	if _, err := s.db.Exec(`DELETE FROM mock_rules WHERE id = ?`, id); err != nil {
		return err
	}
	return s.reload()
}

// reorder puts the listed stored rules first, in that order; the rest keep
// their relative order after them. Config rules cannot be moved.
func (s *mockStore) reorder(ids []string) error {
	s.mu.RLock()
	stored := append([]MockRule(nil), s.stored...)
	s.mu.RUnlock()

	byID := make(map[string]bool, len(stored))
	for _, r := range stored {
		byID[r.ID] = true
	}
	seen := map[string]bool{}
	order := make([]string, 0, len(stored))
	for _, id := range ids {
		if !byID[id] {
			return fmt.Errorf("%w: %s", errMockNotFound, id)
		}
		if !seen[id] {
			seen[id] = true
			order = append(order, id)
		}
	}
	for _, r := range stored {
		if !seen[r.ID] {
			order = append(order, r.ID)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range order {
		if _, err := tx.Exec(`UPDATE mock_rules SET position = ? WHERE id = ?`, i, id); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.reload()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
