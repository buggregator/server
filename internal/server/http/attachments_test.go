package http_test

import (
	"net/http"
	"testing"

	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/storage"
	smtpmod "github.com/buggregator/go-buggregator/modules/smtp"
)

// TestRegisterAttachmentAPI_NoConflictWithSMTPRoutes ensures the attachment
// routes can co-exist on the same mux as the SMTP module's message routes.
//
// Prior layout used `/api/smtp/{eventUuid}/attachments/{uuid}` which collides
// with `/api/smtp/message/{uuid}/raw` (both 4-segment patterns, neither more
// specific), and Go's ServeMux panics at registration time. This test fails
// the same way if the conflict ever returns.
func TestRegisterAttachmentAPI_NoConflictWithSMTPRoutes(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	attachments := storage.NewAttachmentStore("memory", "")
	mux := http.NewServeMux()

	// Order intentionally interleaves attachment + SMTP module routes to
	// mirror app startup.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked: %v", r)
		}
	}()

	serverhttp.RegisterAttachmentAPI(mux, db, attachments)

	mod := smtpmod.New(":0", attachments, db)
	mod.RegisterRoutes(mux, nil)
}
