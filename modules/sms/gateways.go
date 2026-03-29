package sms

// gateway defines how to detect and parse SMS from a specific provider.
type gateway struct {
	Name         string
	DetectFields []string // ALL must be present to detect. Empty = always match (generic).
	FromFields   []string // Candidate field names for "from", first non-empty wins.
	ToFields     []string // Candidate field names for "to".
	MsgFields    []string // Candidate field names for "message".
}

// detect returns true if all DetectFields are present in body.
func (g *gateway) detect(body map[string]any) bool {
	if len(g.DetectFields) == 0 {
		return true // generic fallback
	}
	for _, f := range g.DetectFields {
		if _, ok := body[f]; !ok {
			return false
		}
	}
	return true
}

// validate returns list of missing field group descriptions.
func (g *gateway) validate(body map[string]any) []string {
	var missing []string

	for _, f := range g.DetectFields {
		if _, ok := body[f]; !ok {
			missing = append(missing, f)
		}
	}

	if extractFirst(body, g.FromFields) == "" {
		missing = append(missing, joinFields(g.FromFields))
	}
	if extractFirst(body, g.ToFields) == "" {
		missing = append(missing, joinFields(g.ToFields))
	}
	if extractFirst(body, g.MsgFields) == "" {
		missing = append(missing, joinFields(g.MsgFields))
	}

	return missing
}

// parse extracts from/to/message from body using field mappings.
func (g *gateway) parse(body map[string]any) SmsMessage {
	return SmsMessage{
		From:    extractFirst(body, g.FromFields),
		To:      extractFirst(body, g.ToFields),
		Message: extractFirst(body, g.MsgFields),
		Gateway: g.Name,
	}
}

// allGateways returns all 41+ gateways in detection priority order.
// Most unique detection fields first, generic last.
func allGateways() []*gateway {
	return []*gateway{
		// --- Highly unique detection ---
		{Name: "twilio", DetectFields: []string{"MessageSid", "Body"}, FromFields: []string{"From"}, ToFields: []string{"To"}, MsgFields: []string{"Body"}},
		{Name: "plivo", DetectFields: []string{"MessageUUID"}, FromFields: []string{"From", "src"}, ToFields: []string{"To", "dst"}, MsgFields: []string{"Text", "text"}},
		{Name: "sinch", DetectFields: []string{"batch_id", "body"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"body"}},
		{Name: "vonage", DetectFields: []string{"api_key", "api_secret"}, FromFields: []string{"from", "msisdn"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "infobip", DetectFields: []string{"messages"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "messagebird", DetectFields: []string{"originator", "recipients"}, FromFields: []string{"originator"}, ToFields: []string{"recipients"}, MsgFields: []string{"body"}},
		{Name: "telnyx", DetectFields: []string{"messaging_profile_id"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "bandwidth", DetectFields: []string{"applicationId", "text"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "brevo", DetectFields: []string{"sender", "recipient", "content"}, FromFields: []string{"sender"}, ToFields: []string{"recipient"}, MsgFields: []string{"content"}},
		{Name: "termii", DetectFields: []string{"api_key", "sms"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"sms"}},
		{Name: "smsfactor", DetectFields: []string{"gsmsmsid"}, FromFields: []string{"sender"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "messagemedia", DetectFields: []string{"source_number", "destination_number"}, FromFields: []string{"source_number"}, ToFields: []string{"destination_number"}, MsgFields: []string{"content"}},
		{Name: "lox24", DetectFields: []string{"sender_id", "phone"}, FromFields: []string{"sender_id"}, ToFields: []string{"phone"}, MsgFields: []string{"text"}},
		{Name: "unifonic", DetectFields: []string{"AppSid", "Body", "Recipient"}, FromFields: []string{"SenderID"}, ToFields: []string{"Recipient"}, MsgFields: []string{"Body"}},
		{Name: "smsc", DetectFields: []string{"login", "psw", "phones"}, FromFields: []string{"sender"}, ToFields: []string{"phones"}, MsgFields: []string{"mes"}},
		{Name: "isendpro", DetectFields: []string{"keyid", "num", "sms"}, FromFields: []string{"emetteur"}, ToFields: []string{"num"}, MsgFields: []string{"sms"}},
		{Name: "yunpian", DetectFields: []string{"apikey", "mobile"}, FromFields: []string{"from"}, ToFields: []string{"mobile"}, MsgFields: []string{"text"}},
		{Name: "simpletextin", DetectFields: []string{"contactPhone", "text"}, FromFields: []string{"accountPhone"}, ToFields: []string{"contactPhone"}, MsgFields: []string{"text"}},
		{Name: "sendberry", DetectFields: []string{"key", "name", "content"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"content"}},
		{Name: "turbosms", DetectFields: []string{"sms"}, FromFields: []string{"sender"}, ToFields: []string{"recipients"}, MsgFields: []string{"text"}},
		{Name: "clicksend", DetectFields: []string{"messages"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"body"}},
		{Name: "smsmode", DetectFields: []string{"body", "recipient"}, FromFields: []string{"from"}, ToFields: []string{"recipient"}, MsgFields: []string{"body"}},

		// --- Russian providers ---
		{Name: "smsru", DetectFields: []string{"api_id", "msg"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"msg"}},
		{Name: "smsaero", DetectFields: []string{"sign", "number"}, FromFields: []string{"sign"}, ToFields: []string{"number"}, MsgFields: []string{"text"}},
		{Name: "devino", DetectFields: []string{"SourceAddress", "DestinationAddress", "Data"}, FromFields: []string{"SourceAddress"}, ToFields: []string{"DestinationAddress"}, MsgFields: []string{"Data"}},
		{Name: "iqsms", DetectFields: []string{"clientId", "phone"}, FromFields: []string{"sender"}, ToFields: []string{"phone"}, MsgFields: []string{"text"}},
		{Name: "mts", DetectFields: []string{"naming", "msisdn"}, FromFields: []string{"naming"}, ToFields: []string{"msisdn"}, MsgFields: []string{"text"}},
		{Name: "beeline", DetectFields: []string{"target", "message"}, FromFields: []string{"sender"}, ToFields: []string{"target"}, MsgFields: []string{"message"}},
		{Name: "megafon", DetectFields: []string{"subject", "message"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"message"}},

		// --- Less unique, multi-field discrimination ---
		{Name: "sevenio", DetectFields: []string{"json", "text", "to"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "gatewayapi", DetectFields: []string{"recipients", "message", "sender"}, FromFields: []string{"sender"}, ToFields: []string{"recipients"}, MsgFields: []string{"message"}},
		{Name: "redlink", DetectFields: []string{"phoneNumbers", "message"}, FromFields: []string{"sender"}, ToFields: []string{"phoneNumbers"}, MsgFields: []string{"message"}},
		{Name: "ovhcloud", DetectFields: []string{"receivers", "message"}, FromFields: []string{"sender"}, ToFields: []string{"receivers"}, MsgFields: []string{"message"}},
		{Name: "primotexto", DetectFields: []string{"number", "message", "from"}, FromFields: []string{"from"}, ToFields: []string{"number"}, MsgFields: []string{"message"}},
		{Name: "mobyt", DetectFields: []string{"message_type", "message"}, FromFields: []string{"sender"}, ToFields: []string{"recipient"}, MsgFields: []string{"message"}},
		{Name: "smsapi", DetectFields: []string{"format", "encoding", "message"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"message"}},
		{Name: "octopush", DetectFields: []string{"sms_text", "sms_recipients"}, FromFields: []string{"sms_sender"}, ToFields: []string{"sms_recipients"}, MsgFields: []string{"sms_text"}},
		{Name: "smsbiuras", DetectFields: []string{"uid", "apikey", "message"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"message"}},
		{Name: "46elks", DetectFields: []string{"from", "to", "message"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"message"}},
		{Name: "clickatell", DetectFields: []string{"from", "to", "text"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "allmysms", DetectFields: []string{"from", "to", "text"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},
		{Name: "ringcentral", DetectFields: []string{"from", "to", "text"}, FromFields: []string{"from"}, ToFields: []string{"to"}, MsgFields: []string{"text"}},

		// --- Generic fallback (always matches) ---
		{
			Name:         "generic",
			DetectFields: []string{},
			FromFields:   []string{"from", "From", "sender", "Sender", "originator", "msisdn", "SenderID", "source_number", "sign", "naming", "SourceAddress", "sms_sender", "emetteur", "sender_id"},
			ToFields:     []string{"to", "To", "recipient", "Recipient", "dst", "phone", "phones", "mobile", "num", "destination_number", "contactPhone", "number", "target", "DestinationAddress", "sms_recipients", "receivers", "phoneNumbers"},
			MsgFields:    []string{"message", "body", "Body", "text", "Text", "content", "Content", "sms", "mes", "msg", "Data", "sms_text"},
		},
	}
}

// findByName returns a gateway by slug, or nil.
func findByName(name string) *gateway {
	for _, g := range allGateways() {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// detectGateway returns the first gateway that matches the body.
func detectGatewayFromBody(body map[string]any) *gateway {
	for _, g := range allGateways() {
		if g.detect(body) {
			return g
		}
	}
	return nil
}
