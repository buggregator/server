package sms

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// twilioSendPath is Twilio's Programmable Messaging send endpoint, the one
// SDKs and Symfony's Twilio transport post to.
var twilioSendPath = regexp.MustCompile(`^/2010-04-01/Accounts/([^/]+)/Messages\.json$`)

// twilioAccountSID returns the account SID when path is Twilio's send
// endpoint.
func twilioAccountSID(path string) (string, bool) {
	m := twilioSendPath.FindStringSubmatch(path)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// handleTwilioSend stores a message sent through Twilio's API and answers
// like Twilio: 201 with the message resource, or 400 with a Twilio error when
// a required parameter is missing (the message is still stored, with
// warnings).
func handleTwilioSend(accountSID string, body map[string]any) *event.Incoming {
	gw := findByName("twilio")
	msg := gw.parse(body)
	if msg.From == "" {
		msg.From = extractFirst(body, []string{"MessagingServiceSid"})
	}

	var missing []string
	var twErr *twilioError
	if msg.To == "" {
		missing = append(missing, "To")
		twErr = &twilioError{Code: 21604, Message: "A 'To' phone number is required.", Status: http.StatusBadRequest}
	}
	if msg.From == "" {
		missing = append(missing, "From|MessagingServiceSid")
		if twErr == nil {
			twErr = &twilioError{Code: 21603, Message: "A 'From' phone number is required.", Status: http.StatusBadRequest}
		}
	}
	if msg.Message == "" {
		missing = append(missing, "Body")
		if twErr == nil {
			twErr = &twilioError{Code: 21602, Message: "Message body is required.", Status: http.StatusBadRequest}
		}
	}
	msg.Warnings = missing

	uuid := event.GenerateUUID()
	payload, _ := json.Marshal(msg)
	return &event.Incoming{
		UUID:    uuid,
		Type:    "sms",
		Payload: payload,
		Respond: func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "application/json")
			if twErr != nil {
				twErr.MoreInfo = "https://www.twilio.com/docs/errors/" + strconv.Itoa(twErr.Code)
				w.WriteHeader(twErr.Status)
				_ = json.NewEncoder(w).Encode(twErr)
				return
			}
			sid := "SM" + strings.ReplaceAll(uuid, "-", "")
			now := time.Now().UTC().Format(time.RFC1123Z)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sid":          sid,
				"account_sid":  accountSID,
				"to":           msg.To,
				"from":         msg.From,
				"body":         msg.Message,
				"status":       "queued",
				"direction":    "outbound-api",
				"num_segments": "1",
				"api_version":  "2010-04-01",
				"date_created": now,
				"date_updated": now,
				"uri":          "/2010-04-01/Accounts/" + accountSID + "/Messages/" + sid + ".json",
			})
		},
	}
}

type twilioError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}
