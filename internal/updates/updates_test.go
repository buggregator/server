package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		latest, current string
		want            bool
	}{
		{"2.7.0", "2.6.0", true},
		{"2.10.0", "2.9.3", true},
		{"3.0.0", "2.99.99", true},
		{"2.6.1", "2.6.0", true},
		{"2.6.0", "2.6.0", false},
		{"2.6.0", "2.7.0", false},
		{"2.7.0", "2.7.0-rc.1", true},
		{"2.7.0-rc.2", "2.7.0", false},
		{"v2.7", "2.6.9", true},
		{"2.7.0+build.5", "2.7.0", false},
	} {
		a, okA := parseVersion(tc.latest)
		b, okB := parseVersion(tc.current)
		if !okA || !okB {
			t.Fatalf("%s / %s: parse failed", tc.latest, tc.current)
		}
		if got := newer(a, b); got != tc.want {
			t.Errorf("newer(%s, %s) = %v, want %v", tc.latest, tc.current, got, tc.want)
		}
	}
	for _, bad := range []string{"dev", "", "2.x.0", "1.2.3.4", "latest"} {
		if _, ok := parseVersion(bad); ok {
			t.Errorf("parseVersion(%q) ok", bad)
		}
	}
}

func fakeGitHub(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("User-Agent") == "" {
			t.Error("no User-Agent: GitHub refuses such requests")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestChecker_Check(t *testing.T) {
	srv, _ := fakeGitHub(t, 200, `{"tag_name":"v2.8.0","html_url":"https://github.com/buggregator/server/releases/tag/v2.8.0","body":"x"}`)
	c := New("v2.7.0", 0)
	c.url = srv.URL
	if err := c.check(context.Background()); err != nil {
		t.Fatal(err)
	}
	st := c.Status()
	if st.Current != "2.7.0" || st.Latest != "2.8.0" || !st.Available || st.URL != "https://github.com/buggregator/server/releases/tag/v2.8.0" || st.CheckedAt == nil {
		t.Fatalf("status: %+v", st)
	}

	same, _ := fakeGitHub(t, 200, `{"tag_name":"2.7.0","html_url":"u"}`)
	c.url = same.URL
	if err := c.check(context.Background()); err != nil || c.Status().Available {
		t.Fatalf("same version: %v %+v", err, c.Status())
	}
}

// A failed check keeps the last good result.
func TestChecker_FailureKeepsLast(t *testing.T) {
	good, _ := fakeGitHub(t, 200, `{"tag_name":"v2.8.0","html_url":"u"}`)
	c := New("2.7.0", 0)
	c.url = good.URL
	if err := c.check(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		status int
		body   string
	}{{403, `{"message":"rate limit"}`}, {200, `not json`}, {200, `{"tag_name":"nightly"}`}} {
		bad, _ := fakeGitHub(t, tc.status, tc.body)
		c.url = bad.URL
		if err := c.check(context.Background()); err == nil {
			t.Errorf("%d %s: no error", tc.status, tc.body)
		}
		if st := c.Status(); st.Latest != "2.8.0" || !st.Available {
			t.Errorf("%d %s: lost last result: %+v", tc.status, tc.body, st)
		}
	}
}

// Run checks once at start, waits the retry delay after a failure, and stops
// with the context.
func TestChecker_RunSchedule(t *testing.T) {
	srv, hits := fakeGitHub(t, 500, ``)
	c := New("2.7.0", 0)
	c.url = srv.URL
	c.retry = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { c.Run(ctx); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for hits.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if hits.Load() < 3 {
		t.Fatalf("retries: %d", hits.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}

	// Success: the next check is a full interval away.
	ok, okHits := fakeGitHub(t, 200, `{"tag_name":"v2.7.0","html_url":"u"}`)
	c2 := New("2.7.0", 0)
	c2.url = ok.URL
	c2.retry = time.Millisecond
	ctx2, cancel2 := context.WithCancel(context.Background())
	done2 := make(chan struct{})
	go func() { c2.Run(ctx2); close(done2) }()
	for c2.Status().CheckedAt == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	cancel2()
	<-done2
	if okHits.Load() != 1 {
		t.Fatalf("checks after success: %d, want 1", okHits.Load())
	}
}

// A dev build never calls GitHub.
func TestChecker_DevNoNetwork(t *testing.T) {
	srv, hits := fakeGitHub(t, 200, `{"tag_name":"v2.8.0","html_url":"u"}`)
	c := New("dev", 0)
	c.url = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c.Run(ctx)
	if hits.Load() != 0 || c.Status().Available {
		t.Fatalf("dev build checked: %d %+v", hits.Load(), c.Status())
	}
}

func TestNew_IntervalFloor(t *testing.T) {
	if c := New("2.7.0", time.Minute); c.interval != time.Hour {
		t.Errorf("interval = %v", c.interval)
	}
	if c := New("2.7.0", 0); c.interval != DefaultInterval {
		t.Errorf("default interval = %v", c.interval)
	}
}
