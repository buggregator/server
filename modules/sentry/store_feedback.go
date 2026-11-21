package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// Feedback sources: the envelope item type it came in.
const (
	feedbackSourceUserReport = "user_report"
	feedbackSourceFeedback   = "feedback"
)

// Feedback is one piece of user feedback, normalized from either item type.
type Feedback struct {
	EventID    string // the error it is about; "" when none
	FeedbackID string // the feedback event's own id; "" for user_report
	Name       string
	Email      string
	Message    string
	URL        string
	Source     string
	Payload    json.RawMessage
}

// parseUserReport reads the legacy "user_report" item:
// {"event_id", "name", "email", "comments"}.
func parseUserReport(payload json.RawMessage) (*Feedback, error) {
	var r struct {
		EventID  string `json:"event_id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Comments string `json:"comments"`
	}
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, err
	}
	return &Feedback{
		EventID: r.EventID,
		Name:    r.Name,
		Email:   r.Email,
		Message: r.Comments,
		Source:  feedbackSourceUserReport,
		Payload: payload,
	}, nil
}

// parseFeedbackItem reads the "feedback" item: an event whose
// contexts.feedback carries message, contact_email, name, url and
// associated_event_id. envelopeEventID fills a missing event_id.
func parseFeedbackItem(payload json.RawMessage, envelopeEventID string) (*Feedback, error) {
	var ev struct {
		EventID  string `json:"event_id"`
		Contexts struct {
			Feedback struct {
				Message           string `json:"message"`
				ContactEmail      string `json:"contact_email"`
				Name              string `json:"name"`
				URL               string `json:"url"`
				AssociatedEventID string `json:"associated_event_id"`
			} `json:"feedback"`
		} `json:"contexts"`
		User struct {
			Email    string `json:"email"`
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	fb := ev.Contexts.Feedback
	return &Feedback{
		EventID:    fb.AssociatedEventID,
		FeedbackID: firstNonEmpty(ev.EventID, envelopeEventID),
		Name:       firstNonEmpty(fb.Name, ev.User.Name, ev.User.Username),
		Email:      firstNonEmpty(fb.ContactEmail, ev.User.Email),
		Message:    fb.Message,
		URL:        fb.URL,
		Source:     feedbackSourceFeedback,
		Payload:    payload,
	}, nil
}

// storeFeedback inserts one feedback row. Event ids are kept as sent, so
// they match the error's card uuid.
func storeFeedback(db *sql.DB, fb *Feedback, project string) error {
	_, err := db.Exec(
		`INSERT INTO sentry_feedback
			(id, project_id, event_id, feedback_id, name, email, message, url, source, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.GenerateUUID(),
		projectKey(project),
		strings.TrimSpace(fb.EventID),
		strings.TrimSpace(fb.FeedbackID),
		fb.Name, fb.Email, fb.Message, fb.URL, fb.Source,
		nullIfEmpty(string(fb.Payload)),
	)
	if err != nil {
		return fmt.Errorf("insert sentry_feedback: %w", err)
	}
	return nil
}

// normalizeEventID lowercases an event id and drops its dashes:
// "5A9D...-..." and "5a9d..." name the same event.
func normalizeEventID(id string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), "-", ""))
}
