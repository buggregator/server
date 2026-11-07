package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// --- Ray ------------------------------------------------------------------------

func rayOrigin(file string, line int, host string) map[string]any {
	return map[string]any{"file": file, "line_number": line, "hostname": host}
}

func (s *seeder) sendRay(project string, payloads ...map[string]any) {
	body, _ := json.Marshal(map[string]any{
		"uuid":     uuid4(),
		"payloads": payloads,
		"meta": map[string]any{
			"php_version": "8.3.12", "php_version_id": 80312, "project_name": project,
			"laravel_version": "11.31.0", "laravel_ray_package_version": "1.37.1", "ray_package_version": "1.41.2",
		},
	})
	h := projectHeaders("ray", project)
	h["User-Agent"] = "Ray 1.41.2"
	h["Content-Type"] = "application/json"
	if err := s.post("POST", "/", h, body); err != nil {
		s.fail("ray "+fmt.Sprint(payloads[0]["type"]), err)
		return
	}
	s.count("ray", 1)
}

func (s *seeder) seedRay() {
	shopOrigin := rayOrigin("/var/www/html/app/Http/Controllers/CheckoutController.php", 38, "shop-web-1")
	billOrigin := rayOrigin("/var/www/html/app/Billing/ChargeService.php", 44, "billing-api-1")

	s.sendRay("shop",
		map[string]any{"type": "log", "content": map[string]any{"values": []string{"Checkout started for cart c7f1e2"}}, "origin": shopOrigin},
		map[string]any{"type": "color", "content": map[string]any{"color": "green"}, "origin": shopOrigin},
	)
	s.sendRay("shop",
		map[string]any{"type": "table", "content": map[string]any{"label": "Cart", "values": map[string]any{
			"cart_id": "c7f1e2", "customer": "alex.morgan@example.com", "items": "3",
			"subtotal": "119.90 EUR", "shipping": "10.00 EUR", "total": "129.90 EUR", "coupon": "(none)",
		}}, "origin": shopOrigin},
	)
	s.sendRay("shop",
		map[string]any{"type": "executed_query", "content": map[string]any{
			"sql":             "select * from `products` where `category_id` = ? and `price` between ? and ? order by `price` asc limit 24",
			"bindings":        []any{12, 50, 150},
			"connection_name": "mysql", "time": 14.82,
		}, "origin": rayOrigin("/var/www/html/app/Repositories/ProductRepository.php", 61, "shop-web-1")},
	)
	s.sendRay("shop",
		map[string]any{"type": "measure", "content": map[string]any{
			"name": "price-recalculation", "is_new_timer": false, "total_time": 184, "max_memory_usage_during_total_time": 6291456,
			"time_since_last_call": 92.4, "max_memory_usage_since_last_call": 4194304,
		}, "origin": rayOrigin("/var/www/html/app/Services/PriceCalculator.php", 27, "shop-web-1")},
	)
	s.sendRay("billing",
		map[string]any{"type": "log", "content": map[string]any{"values": []string{"Retrying charge for INV-2026-0042 (attempt 2 of 3)"}}, "origin": billOrigin},
		map[string]any{"type": "label", "content": map[string]any{"label": "dunning"}, "origin": billOrigin},
		map[string]any{"type": "color", "content": map[string]any{"color": "orange"}, "origin": billOrigin},
	)
	s.sendRay("billing",
		map[string]any{"type": "exception", "content": map[string]any{
			"class":   "App\\Billing\\Exceptions\\PaymentDeclinedException",
			"message": "Card declined: insufficient_funds (invoice INV-2026-0042)",
			"frames": []map[string]any{
				{"file_name": "/var/www/html/app/Billing/Gateway/CardGateway.php", "line_number": 74, "class": "App\\Billing\\Gateway\\CardGateway", "method": "charge", "vendor_frame": false,
					"snippet": []map[string]any{
						{"line_number": 72, "text": "        if ($response->json('status') === 'declined') {"},
						{"line_number": 73, "text": "            // the customer gets a dunning email from the listener"},
						{"line_number": 74, "text": "            throw PaymentDeclinedException::fromResponse($response, $invoice);"},
						{"line_number": 75, "text": "        }"},
					}},
				{"file_name": "/var/www/html/app/Billing/ChargeService.php", "line_number": 46, "class": "App\\Billing\\ChargeService", "method": "chargeInvoice", "vendor_frame": false, "snippet": []any{}},
				{"file_name": "/var/www/html/vendor/laravel/framework/src/Illuminate/Queue/CallQueuedHandler.php", "line_number": 130, "class": "Illuminate\\Queue\\CallQueuedHandler", "method": "call", "vendor_frame": true, "snippet": []any{}},
			},
			"meta": []any{},
		}, "origin": billOrigin},
	)
	s.sendRay("billing",
		map[string]any{"type": "notify", "content": map[string]any{"value": "Invoice batch 2026-11 finished: 312 sent, 4 failed"}, "origin": billOrigin},
	)
}

// --- Inspector ---------------------------------------------------------------------

func (s *seeder) seedInspector() {
	for _, t := range []struct {
		project, name, host string
		dur                 float64
		segments            [][3]any // type, label, duration ms
	}{
		{"shop", "GET /products/{slug}", "shop-web-1", 92.4, [][3]any{
			{"mysql", "select * from `products` where `slug` = ? limit 1", 7.9},
			{"mysql", "select * from `product_images` where `product_id` in (?)", 2.8},
			{"redis", "GET product:trail-runner-x", 0.6},
			{"view", "products.show", 41.2},
		}},
		{"shop", "POST /checkout", "shop-web-1", 388.1, [][3]any{
			{"mysql", "insert into `orders` (`number`, `customer_id`, `total`) values (?, ?, ?)", 5.3},
			{"http", "POST https://billing.example.com/api/v1/charges", 286.0},
			{"job", "App\\Jobs\\SendOrderConfirmation", 3.1},
		}},
		{"billing", "App\\Jobs\\GenerateInvoicePdf", "billing-worker-1", 612.7, [][3]any{
			{"pgsql", "select * from invoices where id = $1", 2.1},
			{"process", "render pdf.invoice", 402.5},
			{"filesystem", "write invoices/2026/11/INV-2026-0042.pdf", 171.0},
		}},
	} {
		start := s.ago(rnd(2, 40))
		hash := randHex(32)
		hostInfo := map[string]any{"hostname": t.host, "ip": "10.20.0." + fmt.Sprint(rnd(10, 60)), "os": "Linux"}
		txType := "request"
		if strings.HasPrefix(t.name, "App\\") {
			txType = "job"
		}
		txn := map[string]any{
			"model": "transaction", "name": t.name, "type": txType, "hash": hash, "host": hostInfo,
			"timestamp": epoch(start), "memory_peak": float64(rnd(600, 2400)) / 100, "duration": t.dur,
			"result": "success",
		}
		if txType == "request" {
			method, path, _ := strings.Cut(t.name, " ")
			txn["http"] = map[string]any{
				"request": map[string]any{"method": method, "version": "1.1", "socket": map[string]any{"remote_address": "203.0.113.24"},
					"headers": map[string]any{"Host": "shop.example.com", "User-Agent": browsers[0], "Accept": "text/html"}},
				"url": map[string]any{"protocol": "https", "port": "443", "path": path, "search": "", "full": "https://shop.example.com" + path},
			}
			txn["result"] = "200"
		}
		items := []map[string]any{txn}
		offset := 3.0
		for _, sg := range t.segments {
			d := sg[2].(float64)
			items = append(items, map[string]any{
				"model": "segment", "type": sg[0], "label": sg[1], "host": hostInfo,
				"transaction": map[string]any{"name": t.name, "hash": hash, "timestamp": epoch(start)},
				"start":       offset, "timestamp": epoch(start.Add(time.Duration(offset * float64(time.Millisecond)))),
				"duration": d,
			})
			if sg[0] == "mysql" || sg[0] == "pgsql" {
				items[len(items)-1]["context"] = map[string]any{"db": map[string]any{"connection": sg[0], "query": sg[1], "bindings": []any{}}}
			}
			offset += d + 1.5
		}
		raw, _ := json.Marshal(items)
		h := projectHeaders("inspector", t.project)
		h["X-Inspector-Key"] = "demo-ingestion-key"
		h["X-Inspector-Version"] = "4.8.2"
		h["Content-Type"] = "application/json"
		if err := s.post("POST", "/", h, []byte(base64.StdEncoding.EncodeToString(raw))); err != nil {
			s.fail("inspector "+t.name, err)
			continue
		}
		s.count("inspector", 1)
	}
}

// --- SMS ----------------------------------------------------------------------------

func (s *seeder) seedSMS() {
	form := func(v url.Values) ([]byte, string) { return []byte(v.Encode()), "application/x-www-form-urlencoded" }
	js := func(v map[string]any) ([]byte, string) { b, _ := json.Marshal(v); return b, "application/json" }

	type msg struct {
		path string
		body []byte
		ct   string
	}
	var msgs []msg
	add := func(path string, b []byte, ct string) { msgs = append(msgs, msg{path, b, ct}) }

	b, ct := form(url.Values{"MessageSid": {"SM" + randHex(16)}, "From": {"+15550100200"}, "To": {"+15550142233"},
		"Body": {"Your Example Shop order ORD-2026-11-48213 has shipped. Track it: https://shop.example.com/t/48213"}})
	add("/sms/twilio/shop", b, ct)
	b, ct = form(url.Values{"MessageSid": {"SM" + randHex(16)}, "From": {"+15550100200"}, "To": {"+15550187711"},
		"Body": {"Example Shop: your login code is 482913. It expires in 10 minutes."}})
	add("/sms/twilio/shop", b, ct)
	b, ct = js(map[string]any{"originator": "ExampleShop", "recipients": "+15550163390", "body": "Good news! Trail Runner X in size 42 is back in stock."})
	add("/sms/messagebird/shop", b, ct)
	b, ct = js(map[string]any{"api_key": "demo", "api_secret": "demo", "from": "Billing", "to": "+15550129944",
		"text": "Payment of 49.00 EUR for invoice INV-2026-0042 failed. Update your card at https://billing.example.com/pay"})
	add("/sms/vonage/billing", b, ct)
	b, ct = js(map[string]any{"from": "+15550100300", "to": "+15550152020", "text": "Invoice INV-2026-0057 (129.00 EUR) is due on Nov 15."})
	add("/sms/clickatell/billing", b, ct)

	for _, m := range msgs {
		if err := s.post("POST", m.path, map[string]string{"Content-Type": m.ct}, m.body); err != nil {
			s.fail("sms "+m.path, err)
			continue
		}
		s.count("sms", 1)
	}
}

// --- HTTP dumps ------------------------------------------------------------------------

func (s *seeder) seedHTTPDumps() {
	jsonBody := func(v any) []byte { b, _ := json.Marshal(v); return b }
	reqs := []struct {
		project, method, path string
		headers               map[string]string
		body                  []byte
	}{
		{"shop", "GET", "/v1/products?category=running&size=42&sort=price_asc&page=2", map[string]string{"Accept": "application/json", "Authorization": "Bearer demo-token-shop"}, nil},
		{"shop", "PUT", "/v1/cart/items/42", map[string]string{"Content-Type": "application/json", "Cookie": "shop_session=9f2c1a; locale=en"},
			jsonBody(map[string]any{"sku": "SKU-1042", "size": "42", "quantity": 2})},
		{"shop", "POST", "/v1/newsletter/subscribe", map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			[]byte("email=jamie.chen%40example.com&source=footer&consent=1")},
		{"shop", "DELETE", "/v1/sessions/current", map[string]string{"Authorization": "Bearer demo-token-shop"}, nil},
		{"billing", "POST", "/webhooks/payments", map[string]string{"Content-Type": "application/json", "X-Signature": "t=" + fmt.Sprint(s.now.Unix()) + ",v1=" + randHex(32)},
			jsonBody(map[string]any{"id": "evt_" + randHex(8), "type": "charge.succeeded", "created": s.now.Unix(),
				"data": map[string]any{"charge": "ch_" + randHex(8), "amount": 4900, "currency": "eur", "invoice": "INV-2026-0042"}})},
		{"billing", "PATCH", "/v1/subscriptions/sub_7781?notify=false", map[string]string{"Content-Type": "application/json"},
			jsonBody(map[string]any{"plan": "pro-annual", "seats": 12, "proration": "create_invoice"})},
	}
	for _, r := range reqs {
		h := projectHeaders("http-dump", r.project)
		h["User-Agent"] = "Example-Client/3.2 (+https://example.com)"
		for k, v := range r.headers {
			h[k] = v
		}
		if err := s.post(r.method, r.path, h, r.body); err != nil {
			s.fail("http-dump "+r.method+" "+r.path, err)
			continue
		}
		s.count("http-dump", 1)
	}

	// A multipart upload with two files.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("period", "2026-10")
	_ = mw.WriteField("uploaded_by", "finance@example.com")
	addFile := func(field, name, ct string, data []byte) {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, name))
		h.Set("Content-Type", ct)
		w, _ := mw.CreatePart(h)
		_, _ = w.Write(data)
	}
	addFile("statement", "statement-2026-10.csv", "text/csv", []byte(invoiceCSV))
	addFile("receipt", "receipt-INV-2026-0042.pdf", "application/pdf", demoPDF("Receipt INV-2026-0042 - 49.00 EUR paid"))
	_ = mw.Close()
	h := projectHeaders("http-dump", "billing")
	h["Content-Type"] = mw.FormDataContentType()
	if err := s.post("POST", "/v1/statements/upload", h, buf.Bytes()); err != nil {
		s.fail("http-dump upload", err)
	} else {
		s.count("http-dump", 1)
	}
}

const invoiceCSV = `invoice,customer,amount,currency,status,due_date
INV-2026-0040,acct_7781,129.00,EUR,paid,2026-10-05
INV-2026-0041,acct_9012,49.00,EUR,paid,2026-10-12
INV-2026-0042,acct_9012,49.00,EUR,failed,2026-10-19
INV-2026-0043,acct_3345,1290.00,EUR,open,2026-11-02
`

// demoPDF returns a tiny one-page PDF showing a line of text.
func demoPDF(text string) []byte {
	stream := fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", text)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

// --- Mock server ----------------------------------------------------------------------

type mockRule struct {
	Project string            `json:"project"`
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	DelayMs int               `json:"delay_ms"`
}

func (s *seeder) seedMocks() {
	jsonHdr := map[string]string{"Content-Type": "application/json"}
	rules := []mockRule{
		{Project: "shop", Name: "Payment status", Method: "GET", Path: "/payments/{id}", Status: 200, Headers: jsonHdr,
			Body: `{"id":"pay_7Hq2","status":"succeeded","amount":12990,"currency":"EUR"}`},
		{Project: "shop", Name: "Create payment", Method: "POST", Path: "/payments", Status: 201, Headers: jsonHdr,
			Body: `{"id":"pay_9Kd4","status":"requires_capture"}`},
		{Project: "shop", Name: "Shipping rates", Method: "GET", Path: "/shipping/rates", Status: 200, Headers: jsonHdr, DelayMs: 250,
			Body: `{"rates":[{"carrier":"Parcel Express","service":"standard","price":4.90,"days":3},{"carrier":"Parcel Express","service":"next_day","price":12.50,"days":1}]}`},
		{Project: "billing", Name: "Tax calculation", Method: "POST", Path: "/tax/calculate", Status: 200, Headers: jsonHdr,
			Body: `{"country":"DE","rate":0.19,"tax":9.31,"total":58.31}`},
		{Project: "billing", Name: "FX rates (outage)", Method: "GET", Path: "/fx/rates", Status: 503,
			Headers: map[string]string{"Content-Type": "application/json", "Retry-After": "30"}, Body: `{"error":"upstream_unavailable"}`},
		{Project: "", Name: "Health check", Method: "ANY", Path: "/health", Status: 200, Headers: jsonHdr, Body: `{"status":"ok"}`},
	}

	// Re-runs reuse the rules created before.
	existing := map[string]bool{}
	if out, err := s.do("GET", "/api/mocks", nil, nil, true); err == nil {
		var list struct {
			Data []mockRule `json:"data"`
		}
		if json.Unmarshal(out, &list) == nil {
			for _, r := range list.Data {
				existing[r.Project+" "+r.Method+" "+r.Path] = true
			}
		}
	} else {
		s.fail("list mock rules", err)
	}
	for _, r := range rules {
		if existing[r.Project+" "+r.Method+" "+r.Path] {
			s.count("mock rules (reused)", 1)
			continue
		}
		body, _ := json.Marshal(r)
		if err := s.post("POST", "/api/mocks", jsonHdr, body); err != nil {
			s.fail("mock rule "+r.Name, err)
			continue
		}
		s.count("mock rules", 1)
	}

	hits := []struct {
		project, method, path string
		body                  string
	}{
		{"shop", "GET", "/payments/pay_7Hq2", ""},
		{"shop", "GET", "/payments/pay_3Lm8", ""},
		{"shop", "POST", "/payments", `{"order":"ORD-2026-11-48213","amount":12990,"currency":"EUR"}`},
		{"shop", "GET", "/shipping/rates?country=DE&weight=1.2", ""},
		{"shop", "GET", "/health", ""},
		{"billing", "POST", "/tax/calculate", `{"country":"DE","amount":49.00}`},
		{"billing", "GET", "/fx/rates?base=EUR", ""},
		{"billing", "GET", "/fx/rates?base=USD", ""},
		{"billing", "GET", "/refunds/rf_001", ""}, // no rule: answered 404 and still captured
	}
	for _, h := range hits {
		hdr := map[string]string{"X-Buggregator-Project": h.project, "Accept": "application/json"}
		var body []byte
		if h.body != "" {
			body = []byte(h.body)
			hdr["Content-Type"] = "application/json"
		}
		if _, err := s.do(h.method, "/mock"+h.path, hdr, body, false); err != nil {
			s.fail("mock hit "+h.path, err)
			continue
		}
		s.count("mock hits", 1)
	}
}
