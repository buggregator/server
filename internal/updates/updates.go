// Package updates asks GitHub for the latest Buggregator server release so the
// UI can say that a new version is out.
//
// One GET of releases/latest at start and then every interval (12 h by
// default); a failed check waits an hour, never less. The check is optional:
// it never blocks start, only logs at debug level, and makes no network call
// when switched off or when the running build has no release version (dev).
package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LatestURL is the GitHub API for the newest published (non-draft,
// non-prerelease) server release.
const LatestURL = "https://api.github.com/repos/buggregator/server/releases/latest"

const (
	DefaultInterval = 12 * time.Hour
	minInterval     = time.Hour
	retryAfter      = time.Hour
	requestTimeout  = 10 * time.Second
	maxBody         = 1 << 20
)

// Status is what the UI reads.
type Status struct {
	Current   string     `json:"current"`
	Latest    string     `json:"latest,omitempty"`     // newest release, without a leading v
	URL       string     `json:"url,omitempty"`        // its release page
	Available bool       `json:"available"`            // Latest is newer than Current
	CheckedAt *time.Time `json:"checked_at,omitempty"` // last successful check
}

// Checker keeps the result of the last successful check.
type Checker struct {
	current  string
	url      string
	interval time.Duration
	retry    time.Duration
	client   *http.Client

	mu sync.RWMutex
	st Status
}

// New returns a checker for the running version. interval <= 0 means the
// default; anything below an hour is raised to an hour.
func New(current string, interval time.Duration) *Checker {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if interval < minInterval {
		interval = minInterval
	}
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	return &Checker{
		current:  current,
		url:      LatestURL,
		interval: interval,
		retry:    retryAfter,
		client:   &http.Client{Timeout: requestTimeout},
		st:       Status{Current: current},
	}
}

// Status returns the last known result.
func (c *Checker) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.st
}

// Run checks at once and then on schedule until ctx ends. A build without a
// release version (dev) never checks.
func (c *Checker) Run(ctx context.Context) {
	if _, ok := parseVersion(c.current); !ok {
		slog.Debug("update check skipped: not a release build", "version", c.current)
		return
	}
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		next := c.interval
		if err := c.check(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Debug("update check failed", "err", err)
			next = c.retry
		}
		t.Reset(next)
	}
}

func (c *Checker) check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "buggregator/"+c.current)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github: %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&rel); err != nil {
		return fmt.Errorf("github: %w", err)
	}
	latest := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	lv, ok := parseVersion(latest)
	if !ok {
		return fmt.Errorf("github: tag %q is not a version", rel.TagName)
	}
	cv, _ := parseVersion(c.current)
	now := time.Now().UTC()

	c.mu.Lock()
	c.st = Status{Current: c.current, Latest: latest, URL: rel.HTMLURL, Available: newer(lv, cv), CheckedAt: &now}
	c.mu.Unlock()
	return nil
}

// version is major.minor.patch plus a pre-release flag ("2.7.0-rc.1").
type version struct {
	n   [3]int
	pre bool
}

func parseVersion(s string) (version, bool) {
	var v version
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], true
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v.n[i] = n
	}
	return v, true
}

// newer reports a > b; a release is newer than its own pre-release.
func newer(a, b version) bool {
	for i := range a.n {
		if a.n[i] != b.n[i] {
			return a.n[i] > b.n[i]
		}
	}
	return b.pre && !a.pre
}
