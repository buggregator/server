package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/buggregator/server/v2/internal/module"
	serverhttp "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/server/ws"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/httpdumps"
	"github.com/buggregator/server/v2/modules/sms"
)

func smsPipeline(t *testing.T) (*serverhttp.IngestionPipeline, *spyStore) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := storage.NewMigrator(db).AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	registry := module.NewRegistry()
	registry.Register(sms.New())
	registry.Register(httpdumps.New(nil, db))
	store := &spyStore{}
	if err := registry.Init(db, http.NewServeMux(), store); err != nil {
		t.Fatal(err)
	}
	es := serverhttp.NewEventService(store, ws.NewHub(), registry, nil, db)
	return serverhttp.NewIngestionPipeline(registry.Handlers(), es, fstest.MapFS{"index.html": {Data: []byte("<html></html>")}}), store
}

// twilioSend builds the request Symfony's Twilio transport sends: form body,
// Basic Auth SID:TOKEN, no Buggregator userinfo.
func twilioSend(form url.Values) *http.Request {
	r := httptest.NewRequest("POST", "http://127.0.0.1:8000/2010-04-01/Accounts/AC123/Messages.json", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth("AC123", "secret-token")
	return r
}

func TestPipeline_TwilioSendAPI(t *testing.T) {
	t.Run("stored as sms, 201 with sid", func(t *testing.T) {
		pipeline, store := smsPipeline(t)
		w := httptest.NewRecorder()
		pipeline.ServeHTTP(w, twilioSend(url.Values{"From": {"+111"}, "To": {"+222"}, "Body": {"Your code is 1234"}}))

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		sid, _ := resp["sid"].(string)
		if !strings.HasPrefix(sid, "SM") || len(sid) != 34 {
			t.Errorf("sid = %q, want SM + 32 hex", sid)
		}
		if resp["account_sid"] != "AC123" || resp["status"] != "queued" || resp["to"] != "+222" {
			t.Errorf("resp = %v", resp)
		}

		evs := store.storedEvents()
		if len(evs) != 1 || evs[0].Type != "sms" {
			t.Fatalf("stored = %+v, want one sms event", evs)
		}
		if evs[0].Project != "default" {
			t.Errorf("project = %q, want default (the auth token is no project)", evs[0].Project)
		}
		if strings.ReplaceAll(evs[0].UUID, "-", "") != strings.TrimPrefix(sid, "SM") {
			t.Errorf("sid %q does not match event %q", sid, evs[0].UUID)
		}
		var msg sms.SmsMessage
		_ = json.Unmarshal(evs[0].Payload, &msg)
		if msg.Gateway != "twilio" || msg.From != "+111" || msg.Message != "Your code is 1234" || len(msg.Warnings) != 0 {
			t.Errorf("payload = %+v", msg)
		}
	})

	t.Run("missing Body: 400 Twilio error, still stored with warning", func(t *testing.T) {
		pipeline, store := smsPipeline(t)
		w := httptest.NewRecorder()
		pipeline.ServeHTTP(w, twilioSend(url.Values{"From": {"+111"}, "To": {"+222"}}))

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["code"] != float64(21602) {
			t.Errorf("resp = %v, want code 21602", resp)
		}
		evs := store.storedEvents()
		if len(evs) != 1 {
			t.Fatalf("stored %d events, want 1", len(evs))
		}
		var msg sms.SmsMessage
		_ = json.Unmarshal(evs[0].Payload, &msg)
		if len(msg.Warnings) != 1 || msg.Warnings[0] != "Body" {
			t.Errorf("warnings = %v", msg.Warnings)
		}
	})

	t.Run("MessagingServiceSid stands in for From", func(t *testing.T) {
		pipeline, store := smsPipeline(t)
		w := httptest.NewRecorder()
		pipeline.ServeHTTP(w, twilioSend(url.Values{"MessagingServiceSid": {"MG1"}, "To": {"+222"}, "Body": {"Hi"}}))
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		var msg sms.SmsMessage
		_ = json.Unmarshal(store.storedEvents()[0].Payload, &msg)
		if msg.From != "MG1" {
			t.Errorf("from = %q, want MG1", msg.From)
		}
	})

	t.Run("other sms requests keep the plain acknowledgement", func(t *testing.T) {
		pipeline, _ := smsPipeline(t)
		r := httptest.NewRequest("POST", "/sms", strings.NewReader(`{"from":"+1","to":"+2","message":"Hi"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		pipeline.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}
