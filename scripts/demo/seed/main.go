// Command seed fills a running Buggregator server with demo data of every
// event type: Sentry (errors, traces, logs, metrics), Ray, Inspector,
// Profiler, SMS, HTTP dumps, mock server rules and hits, SMTP mail and
// Monolog records. VarDumper payloads come from scripts/demo/php instead,
// because they are PHP-serialized objects.
//
// The data belongs to two made-up apps, an online shop ("shop") and a billing
// service ("billing"). Timestamps are relative to now. Running the seed again
// adds more data; mock rules that already exist are reused.
//
//	go run ./scripts/demo/seed -http http://127.0.0.1:8000 -smtp 127.0.0.1:1025 -monolog 127.0.0.1:9913
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

type seeder struct {
	httpBase string
	smtpAddr string
	monolog  string
	client   *http.Client
	now      time.Time

	sent   map[string]int
	order  []string
	failed []string
}

func main() {
	httpBase := flag.String("http", "http://127.0.0.1:8000", "Buggregator HTTP base URL")
	smtpAddr := flag.String("smtp", "127.0.0.1:1025", "Buggregator SMTP address (empty to skip)")
	monolog := flag.String("monolog", "127.0.0.1:9913", "Buggregator Monolog TCP address (empty to skip)")
	flag.Parse()

	s := &seeder{
		httpBase: strings.TrimRight(*httpBase, "/"),
		smtpAddr: *smtpAddr,
		monolog:  *monolog,
		client:   &http.Client{Timeout: 15 * time.Second},
		now:      time.Now(),
		sent:     map[string]int{},
	}

	steps := []struct {
		name string
		fn   func()
	}{
		{"sentry", s.seedSentry},
		{"ray", s.seedRay},
		{"inspector", s.seedInspector},
		{"profiler", s.seedProfiler},
		{"sms", s.seedSMS},
		{"http-dump", s.seedHTTPDumps},
		{"mock", s.seedMocks},
		{"smtp", s.seedSMTP},
		{"monolog", s.seedMonolog},
	}
	for _, st := range steps {
		fmt.Printf("seeding %s...\n", st.name)
		st.fn()
	}

	fmt.Println()
	fmt.Println("sent:")
	for _, k := range s.order {
		fmt.Printf("  %-22s %d\n", k, s.sent[k])
	}
	if len(s.failed) > 0 {
		fmt.Println()
		fmt.Printf("%d request(s) failed:\n", len(s.failed))
		for _, f := range s.failed {
			fmt.Println("  " + f)
		}
		os.Exit(1)
	}
	fmt.Printf("\nOpen %s\n", s.httpBase)
}

// count records one item that reached the server.
func (s *seeder) count(kind string, n int) {
	if _, ok := s.sent[kind]; !ok {
		s.order = append(s.order, kind)
	}
	s.sent[kind] += n
}

func (s *seeder) fail(what string, err error) {
	s.failed = append(s.failed, fmt.Sprintf("%s: %v", what, err))
}

// post sends a request and treats any non-2xx answer as a failure.
func (s *seeder) post(method, path string, headers map[string]string, body []byte) error {
	_, err := s.do(method, path, headers, body, true)
	return err
}

// do sends a request; with strict set, a non-2xx status is an error.
func (s *seeder) do(method, path string, headers map[string]string, body []byte, strict bool) ([]byte, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, s.httpBase+path, rd)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if strict && (resp.StatusCode < 200 || resp.StatusCode > 299) {
		return out, fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, truncate(string(out), 200))
	}
	return out, nil
}

// project headers route an HTTP event of the given type to a project.
func projectHeaders(eventType, project string) map[string]string {
	return map[string]string{
		"X-Buggregator-Event":   eventType,
		"X-Buggregator-Project": project,
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// randHex returns n random bytes as hex (2n characters).
func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// uuid4 returns a random RFC 4122 version 4 UUID.
func uuid4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// rnd returns a random int in [lo, hi].
func rnd(lo, hi int) int {
	if hi <= lo {
		return lo
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo+1)))
	return lo + int(n.Int64())
}

func pick[T any](items []T) T {
	return items[rnd(0, len(items)-1)]
}

// ago returns a time the given number of minutes (plus random seconds) before now.
func (s *seeder) ago(minutes int) time.Time {
	return s.now.Add(-time.Duration(minutes)*time.Minute - time.Duration(rnd(0, 59))*time.Second)
}

// epoch converts a time to fractional Unix seconds, as Sentry SDKs send it.
func epoch(t time.Time) float64 {
	return float64(t.UnixMicro()) / 1e6
}
