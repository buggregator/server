package sentry

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/module"
)

// recordedNotice is one call of handler.notifyStored.
type recordedNotice struct {
	project string
	kinds   []string
}

func recordingHandler(t *testing.T, withDB bool) (*handler, *[]recordedNotice) {
	t.Helper()
	var notices []recordedNotice
	h := &handler{notifyStored: func(project string, kinds []string) {
		notices = append(notices, recordedNotice{project, kinds})
	}}
	if withDB {
		h.db = setupTestDB(t)
	}
	return h, &notices
}

func TestHandleEnvelope_AnnouncesStoredKinds(t *testing.T) {
	logItem := `{"type":"log","item_count":1,"content_type":"application/vnd.sentry.items.log+json"}
{"items":[{"timestamp":1788420877.5,"level":"info","body":"hello"}]}
`
	metricItem := `{"type":"trace_metric","item_count":1,"content_type":"application/vnd.sentry.items.trace-metric+json"}
{"items":[{"timestamp":1788420877.5,"name":"order.status","value":3,"type":"gauge"}]}
`
	transactionItem := `{"type":"transaction"}
{"event_id":"txn-notice","type":"transaction","transaction":"GET /","start_timestamp":1788420877.0,"timestamp":1788420877.5,"contexts":{"trace":{"trace_id":"trace-notice","span_id":"root"}}}
`
	eventItem := `{"type":"event"}
{"event_id":"evt-notice","level":"error","message":"boom"}
`

	tests := []struct {
		name  string
		items string
		want  []string
	}{
		{"metrics only", metricItem, []string{"metric"}},
		{"logs and metrics, logs twice", logItem + metricItem + logItem, []string{"log", "metric"}},
		{"a transaction", transactionItem, []string{"trace"}},
		{"an error event with a log", eventItem + logItem, []string{"log"}},
		{"an error event alone", eventItem, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, notices := recordingHandler(t, true)

			if _, err := h.handleEnvelope([]byte(`{"event_id":"notice"}`+"\n"+tt.items), "proj"); err != nil {
				t.Fatal(err)
			}

			if tt.want == nil {
				if len(*notices) != 0 {
					t.Errorf("notices = %v, want none", *notices)
				}
				return
			}
			if len(*notices) != 1 {
				t.Fatalf("notices = %v, want one", *notices)
			}
			got := (*notices)[0]
			if got.project != "proj" || fmt.Sprint(got.kinds) != fmt.Sprint(tt.want) {
				t.Errorf("notice = %+v, want project proj, kinds %v", got, tt.want)
			}
		})
	}
}

func TestHandleEnvelope_AnnouncesNothingWithoutADatabase(t *testing.T) {
	h, notices := recordingHandler(t, false)

	if _, err := h.handleEnvelope([]byte(traceMetricEnvelope), "proj"); err != nil {
		t.Fatal(err)
	}
	if len(*notices) != 0 {
		t.Errorf("notices = %v, want none: nothing was stored", *notices)
	}
}

func TestHandle_AnnouncesAPlainJSONTransaction(t *testing.T) {
	h, notices := recordingHandler(t, true)

	payload := `{"event_id":"txn-json","type":"transaction","transaction":"GET /","start_timestamp":1788420877.0,"timestamp":1788420877.5,"spans":[],"contexts":{"trace":{"trace_id":"trace-json","span_id":"root"}}}`
	r := httptest.NewRequest("POST", "/api/proj/store", strings.NewReader(payload))

	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc != nil {
		t.Errorf("expected no feed event for a transaction, got %+v", inc)
	}
	if len(*notices) != 1 || fmt.Sprint((*notices)[0].kinds) != "[trace]" {
		t.Errorf("notices = %v, want one with kinds [trace]", *notices)
	}
}

// recordingBroadcaster records what Module.notifyStored sends to the hub.
type recordingBroadcaster struct {
	channel, event string
	payload        any
}

func (b *recordingBroadcaster) Broadcast(channel, eventName string, payload any) {
	b.channel, b.event, b.payload = channel, eventName, payload
}

func TestModule_SendsStoredNoticeOnTheProjectChannel(t *testing.T) {
	var _ module.Broadcasting = (*Module)(nil)

	b := &recordingBroadcaster{}
	m := New()
	m.SetBroadcaster(b)

	m.notifyStored("", []string{"log"})

	if b.channel != "events.project.default" || b.event != StoredEvent {
		t.Errorf("broadcast on %q as %q, want events.project.default as %q", b.channel, b.event, StoredEvent)
	}
	if fmt.Sprint(b.payload) != "map[kinds:[log] project:default]" {
		t.Errorf("payload = %v", b.payload)
	}
}

func TestModule_SendsNothingWithoutABroadcaster(t *testing.T) {
	New().notifyStored("proj", []string{"log"}) // must not panic
}

// A kind is announced only when its rows were stored: a store that fails, an
// item that does not parse or holds nothing announces nothing for that kind.
func TestHandleEnvelope_NoNoticeForFailedStore(t *testing.T) {
	logItem := `{"type":"log"}
{"items":[{"timestamp":1788420877.5,"level":"info","body":"hello"}]}
`
	metricItem := `{"type":"trace_metric"}
{"items":[{"timestamp":1788420877.5,"name":"order.status","value":3,"type":"gauge"}]}
`
	spansItem := `{"type":"spans"}
{"items":[{"trace_id":"trace-fail","span_id":"s1","name":"db","start_timestamp":1788420877.0,"end_timestamp":1788420877.5}]}
`
	tests := []struct {
		name  string
		drop  string // table dropped so its store fails
		items string
		want  []string
	}{
		{"metrics store fails, logs stored", "sentry_metrics", logItem + metricItem, []string{"log"}},
		{"logs store fails", "sentry_logs", logItem, nil},
		{"spans store fails", "sentry_spans", spansItem, nil},
		{"transaction does not parse", "", "{\"type\":\"transaction\"}\n{not json}\n", nil},
		{"empty log items", "", "{\"type\":\"log\"}\n{\"items\":[]}\n", nil},
		{"metric item neither object nor array", "", "{\"type\":\"trace_metric\"}\n\"x\"\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, notices := recordingHandler(t, true)
			if tt.drop != "" {
				if _, err := h.db.Exec("DROP TABLE " + tt.drop); err != nil {
					t.Fatal(err)
				}
			}

			if _, err := h.handleEnvelope([]byte(`{"event_id":"notice"}`+"\n"+tt.items), "proj"); err != nil {
				t.Fatal(err)
			}

			if tt.want == nil {
				if len(*notices) != 0 {
					t.Errorf("notices = %v, want none", *notices)
				}
				return
			}
			if len(*notices) != 1 || fmt.Sprint((*notices)[0].kinds) != fmt.Sprint(tt.want) {
				t.Errorf("notices = %v, want one with kinds %v", *notices, tt.want)
			}
		})
	}
}
