package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeSQL(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"laravel mysql placeholder", "select * from `comments` where `comments`.`post_id` = ? and `comments`.`post_id` is not null",
			"select * from `comments` where `comments`.`post_id` = ? and `comments`.`post_id` is not null"},
		{"inlined number", `select * from "users" where "id" = 42 limit 1`, `select * from "users" where "id" = ? limit ?`},
		{"inlined string with quote", `select * from users where email = 'o''brien@x.io'`, `select * from users where email = ?`},
		{"postgres params", `SELECT * FROM posts WHERE user_id = $1 AND status = $2`, `SELECT * FROM posts WHERE user_id = ? AND status = ?`},
		{"eloquent IN list", "select * from `tags` where `tags`.`id` in (?, ?, ?)", "select * from `tags` where `tags`.`id` IN (?)"},
		{"IN list of literals", "select * from tags where id in (1,2,3,4)", "select * from tags where id IN (?)"},
		{"VALUES rows", "insert into logs (a, b) values (?, ?), (?, ?)", "insert into logs (a, b) VALUES (?)"},
		{"identifier digits kept", `select t1.col2 from t1 where t1.x2fa = 7`, `select t1.col2 from t1 where t1.x2fa = ?`},
		{"whitespace", "select *\n  from   users\twhere id = 1", "select * from users where id = ?"},
		{"decimal", "update p set price = 9.99 where id = 3", "update p set price = ? where id = ?"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := normalizeSQL(c.in); got != c.want {
				t.Errorf("normalizeSQL(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

func TestDetectNPlusOne(t *testing.T) {
	var spans []nPlusOneSpan
	add := func(txn, desc string, ms float64, attrs string) {
		spans = append(spans, nPlusOneSpan{SpanID: fmt.Sprintf("s%d", len(spans)), TransactionID: txn, Description: desc, DurationMS: ms, Attributes: attrs})
	}
	add("t1", "select * from `posts`", 3, "")
	for i := 0; i < 6; i++ {
		attrs := ""
		if i == 2 {
			attrs = `{"code.filepath":"/app/Http/Controllers/PostController.php","code.lineno":27}`
		}
		add("t1", fmt.Sprintf("select * from `comments` where `post_id` = %d", i), 2, attrs)
	}
	// Same query, but only 4 times in another transaction: below threshold.
	for i := 0; i < 4; i++ {
		add("t2", "select * from `comments` where `post_id` = ?", 1, "")
	}
	// 5 times exactly, IN lists of different length.
	for i := 1; i <= 5; i++ {
		ids := "?"
		for j := 1; j < i; j++ {
			ids += ", ?"
		}
		add("t2", "select * from `tags` where `id` in ("+ids+")", 1, `{"code.filepath":"/app/x.php","code.lineno":"9"}`)
	}

	got := detectNPlusOne(spans)
	if len(got) != 2 {
		t.Fatalf("groups = %d, want 2: %v", len(got), got)
	}
	first := got[0]
	if first["count"] != 6 || first["total_ms"] != 12.0 || str(first["transaction_id"]) != "t1" {
		t.Errorf("first group = %v", first)
	}
	if first["query"] != "select * from `comments` where `post_id` = ?" {
		t.Errorf("query = %v", first["query"])
	}
	if str(first["origin"]) != "/app/Http/Controllers/PostController.php:27" {
		t.Errorf("origin = %v", first["origin"])
	}
	if ids := first["span_ids"].([]string); len(ids) != 6 || ids[0] != "s1" {
		t.Errorf("span_ids = %v", ids)
	}
	second := got[1]
	if second["count"] != 5 || str(second["origin"]) != "/app/x.php:9" || second["query"] != "select * from `tags` where `id` IN (?)" {
		t.Errorf("second group = %v", second)
	}

	if got := detectNPlusOne(nil); got == nil || len(got) != 0 {
		t.Errorf("no spans: %v, want empty non-nil", got)
	}
}

func str(v any) string {
	if p, ok := v.(*string); ok && p != nil {
		return *p
	}
	return fmt.Sprint(v)
}

// TestTraceDetail_NPlusOne_Laravel replays a request sentry-laravel 4.31.0
// sent for a page that loads each post's author and comment count in a loop (30 posts)
// (testdata/sdk/laravel-n1, SENTRY_TRACE_SQL_ORIGIN_THRESHOLD_MS=0).
func TestTraceDetail_NPlusOne_Laravel(t *testing.T) {
	db, _ := ingestSDK(t, "laravel-n1")
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)

	const traceID = "919e328d8a5648e58a1c18208d769367"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/traces/"+traceID, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		NPlusOne []struct {
			TransactionID *string  `json:"transaction_id"`
			Query         string   `json:"query"`
			Count         int      `json:"count"`
			TotalMS       float64  `json:"total_ms"`
			Origin        *string  `json:"origin"`
			SpanIDs       []string `json:"span_ids"`
		} `json:"n_plus_one"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		`select * from "users" where "users"."id" = ? limit ?`:                                                             true,
		`select count(*) as aggregate from "comments" where "comments"."post_id" = ? and "comments"."post_id" is not null`: true,
	}
	if len(resp.NPlusOne) != len(want) {
		t.Fatalf("n_plus_one = %+v, want %d groups", resp.NPlusOne, len(want))
	}
	for _, g := range resp.NPlusOne {
		if !want[g.Query] {
			t.Errorf("unexpected group %q", g.Query)
		}
		if g.Count != 30 || len(g.SpanIDs) != 30 {
			t.Errorf("%q: count %d, span_ids %d, want 30", g.Query, g.Count, len(g.SpanIDs))
		}
		if g.TotalMS <= 0 {
			t.Errorf("%q: total_ms %v, want > 0 (sub-millisecond spans summed precisely)", g.Query, g.TotalMS)
		}
		if g.TransactionID == nil {
			t.Errorf("%q: no transaction_id", g.Query)
		}
		// The SDK resolves the origin only within 20 stack frames; lazy
		// relation queries are deeper, so it sends none.
		if g.Origin != nil {
			t.Errorf("%q: origin %q, want none", g.Query, *g.Origin)
		}
	}
}
