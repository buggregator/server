package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// app describes one of the two demo applications as a Sentry SDK sees it.
type app struct {
	project string // Buggregator project key (DSN path)
	service string // service.name on spans: a node on the service map
	host    string // public host name
	server  string // server_name
	release string
	users   []map[string]any
}

var (
	shopApp = app{
		project: "shop",
		service: "shop-web",
		host:    "shop.example.com",
		server:  "shop-web-1",
		release: "shop@2.14.0",
		users: []map[string]any{
			{"id": "1042", "email": "alex.morgan@example.com", "username": "alex.morgan", "ip_address": "203.0.113.24"},
			{"id": "1187", "email": "jamie.chen@example.com", "username": "jamie.chen", "ip_address": "203.0.113.57"},
			{"id": "2203", "email": "sam.patel@example.com", "username": "sam.patel", "ip_address": "198.51.100.12"},
			{"id": "2419", "email": "riley.novak@example.com", "username": "riley.novak", "ip_address": "198.51.100.88"},
		},
	}
	billingApp = app{
		project: "billing",
		service: "billing-api",
		host:    "billing.example.com",
		server:  "billing-api-1",
		release: "billing@1.8.3",
		users: []map[string]any{
			{"id": "acct_7781", "email": "finance@example.com", "username": "finance-team"},
			{"id": "acct_9012", "email": "taylor.brooks@example.com", "username": "taylor.brooks"},
			{"id": "acct_3345", "email": "morgan.lee@example.com", "username": "morgan.lee"},
		},
	}
)

var sentrySDK = map[string]any{
	"name":    "sentry.php.laravel",
	"version": "4.10.1",
	"packages": []map[string]any{
		{"name": "composer:sentry/sentry-laravel", "version": "4.10.1"},
		{"name": "composer:sentry/sentry", "version": "4.10.0"},
	},
}

// --- envelopes ---------------------------------------------------------------

type envItem struct {
	typ     string
	payload any
	extra   map[string]any // extra item header fields (content_type, item_count)
}

// sendEnvelope posts one Sentry envelope to /api/<project>/envelope/.
func (s *seeder) sendEnvelope(project, eventID string, items ...envItem) error {
	var buf bytes.Buffer
	header := map[string]any{
		"sent_at": time.Now().UTC().Format(time.RFC3339),
		"dsn":     s.httpBase + "/" + project,
		"sdk":     map[string]any{"name": sentrySDK["name"], "version": sentrySDK["version"]},
	}
	if eventID != "" {
		header["event_id"] = eventID
	}
	writeJSONLine(&buf, header)
	for _, it := range items {
		body, err := json.Marshal(it.payload)
		if err != nil {
			return err
		}
		ih := map[string]any{"type": it.typ, "length": len(body)}
		for k, v := range it.extra {
			ih[k] = v
		}
		writeJSONLine(&buf, ih)
		buf.Write(body)
		buf.WriteByte('\n')
	}
	return s.post("POST", "/api/"+project+"/envelope/", map[string]string{
		"Content-Type":  "application/x-sentry-envelope",
		"X-Sentry-Auth": "Sentry sentry_version=7, sentry_client=sentry.php.laravel/4.10.1, sentry_key=demo",
		"User-Agent":    "sentry.php.laravel/4.10.1",
	}, buf.Bytes())
}

func writeJSONLine(buf *bytes.Buffer, v any) {
	b, _ := json.Marshal(v)
	buf.Write(b)
	buf.WriteByte('\n')
}

// --- errors ------------------------------------------------------------------

type frame struct {
	file  string
	fn    string
	line  int
	inApp bool
	pre   []string
	code  string
	post  []string
}

func (f frame) toJSON() map[string]any {
	m := map[string]any{
		"filename": strings.TrimPrefix(f.file, "/var/www/html"),
		"abs_path": f.file,
		"function": f.fn,
		"lineno":   f.line,
		"in_app":   f.inApp,
	}
	if f.code != "" {
		m["context_line"] = f.code
		m["pre_context"] = orEmpty(f.pre)
		m["post_context"] = orEmpty(f.post)
	}
	return m
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// vendor builds a framework frame without source context.
func vendor(file, fn string, line int) frame {
	return frame{file: "/var/www/html/vendor/" + file, fn: fn, line: line}
}

// laravelHTTP is the bottom of every Laravel HTTP stack trace (outermost first).
var laravelHTTP = []frame{
	{file: "/var/www/html/public/index.php", fn: "{main}", line: 51, inApp: true,
		pre:  []string{"$kernel = $app->make(Kernel::class);", ""},
		code: "$response = $kernel->handle(",
		post: []string{"    $request = Request::capture()", ")->send();"}},
	vendor("laravel/framework/src/Illuminate/Foundation/Http/Kernel.php", "Illuminate\\Foundation\\Http\\Kernel::handle", 145),
	vendor("laravel/framework/src/Illuminate/Foundation/Http/Kernel.php", "Illuminate\\Foundation\\Http\\Kernel::sendRequestThroughRouter", 176),
	vendor("laravel/framework/src/Illuminate/Pipeline/Pipeline.php", "Illuminate\\Pipeline\\Pipeline::then", 127),
	vendor("laravel/framework/src/Illuminate/Routing/Router.php", "Illuminate\\Routing\\Router::dispatchToRoute", 740),
	vendor("laravel/framework/src/Illuminate/Routing/Route.php", "Illuminate\\Routing\\Route::runController", 260),
	vendor("laravel/framework/src/Illuminate/Routing/ControllerDispatcher.php", "Illuminate\\Routing\\ControllerDispatcher::dispatch", 46),
}

func stack(app []frame) []map[string]any {
	all := append(append([]frame{}, laravelHTTP...), app...)
	out := make([]map[string]any, len(all))
	for i, f := range all {
		out[i] = f.toJSON()
	}
	return out
}

type crumb struct {
	typ, category, level, message string
	data                          map[string]any
}

type errorSpec struct {
	excType, value string
	level          string
	handled        bool
	method, path   string // request
	route          string // transaction name
	body           map[string]any
	frames         []frame
	tags           map[string]string
	extra          map[string]any
	crumbs         []crumb
	times          int
}

func (s *seeder) sendError(a app, e errorSpec, at time.Time, traceID string) error {
	eventID := randHex(16)
	user := pick(a.users)

	crumbs := make([]map[string]any, len(e.crumbs))
	for i, c := range e.crumbs {
		ts := at.Add(-time.Duration(len(e.crumbs)-i) * 35 * time.Millisecond)
		m := map[string]any{"type": c.typ, "category": c.category, "level": c.level, "message": c.message, "timestamp": epoch(ts)}
		if c.data != nil {
			m["data"] = c.data
		}
		crumbs[i] = m
	}

	tags := map[string]string{"route": e.route, "php.version": "8.3.12", "region": "eu-west"}
	for k, v := range e.tags {
		tags[k] = v
	}

	headers := map[string][]string{
		"host":            {a.host},
		"user-agent":      {pick(browsers)},
		"accept":          {"application/json"},
		"accept-language": {"en-US,en;q=0.9"},
		"x-request-id":    {uuid4()},
	}
	req := map[string]any{
		"url":     "https://" + a.host + e.path,
		"method":  e.method,
		"headers": headers,
	}
	if e.body != nil {
		headers["content-type"] = []string{"application/json"}
		req["data"] = e.body
	}

	ev := map[string]any{
		"event_id":    eventID,
		"timestamp":   epoch(at),
		"platform":    "php",
		"level":       e.level,
		"logger":      "laravel",
		"transaction": e.route,
		"server_name": a.server,
		"release":     a.release,
		"environment": "production",
		"sdk":         sentrySDK,
		"tags":        tags,
		"user":        user,
		"request":     req,
		"contexts": map[string]any{
			"os":      map[string]any{"name": "Linux", "version": "6.8.0", "kernel_version": "Linux " + a.server + " 6.8.0-45-generic x86_64"},
			"runtime": map[string]any{"name": "php", "sapi": "fpm-fcgi", "version": "8.3.12"},
			"trace":   map[string]any{"trace_id": traceID, "span_id": randHex(8), "op": "http.server", "status": "internal_error"},
		},
		"breadcrumbs": map[string]any{"values": crumbs},
		"exception": map[string]any{"values": []map[string]any{{
			"type":       e.excType,
			"value":      e.value,
			"mechanism":  map[string]any{"type": "generic", "handled": e.handled},
			"stacktrace": map[string]any{"frames": stack(e.frames)},
		}}},
	}
	if e.extra != nil {
		ev["extra"] = e.extra
	}
	return s.sendEnvelope(a.project, eventID, envItem{typ: "event", payload: ev, extra: map[string]any{"content_type": "application/json"}})
}

var browsers = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
}

func shopErrors() []errorSpec {
	return []errorSpec{
		{
			excType: "Illuminate\\Database\\QueryException",
			value:   "SQLSTATE[23000]: Integrity constraint violation: 1062 Duplicate entry 'ORD-2026-10-48213' for key 'orders.orders_number_unique' (Connection: mysql, SQL: insert into `orders` (`number`, `customer_id`, `total`, `currency`) values (ORD-2026-10-48213, 1042, 129.90, EUR))",
			level:   "error", method: "POST", path: "/checkout", route: "POST /checkout",
			body: map[string]any{"cart_id": "c7f1e2", "shipping_method": "express", "payment_method": "card"},
			frames: []frame{
				vendor("laravel/framework/src/Illuminate/Database/Eloquent/Model.php", "Illuminate\\Database\\Eloquent\\Model::save", 1093),
				{file: "/var/www/html/app/Services/Checkout/OrderFactory.php", fn: "App\\Services\\Checkout\\OrderFactory::create", line: 58, inApp: true,
					pre:  []string{"        $order->number = $this->numbers->next();", "        $order->customer()->associate($cart->customer);", "        $order->total = $cart->total();"},
					code: "        $order->save();",
					post: []string{"", "        foreach ($cart->items as $item) {", "            $order->items()->create($item->toOrderLine());"}},
				{file: "/var/www/html/app/Http/Controllers/CheckoutController.php", fn: "App\\Http\\Controllers\\CheckoutController::store", line: 41, inApp: true,
					pre:  []string{"    public function store(CheckoutRequest $request): RedirectResponse", "    {", "        $cart = $this->carts->current($request->user());"},
					code: "        $order = $this->orders->create($cart, $request->validated());",
					post: []string{"", "        PlaceOrder::dispatch($order);", "        return redirect()->route('orders.show', $order);"}},
			},
			tags: map[string]string{"checkout.step": "place_order"},
			crumbs: []crumb{
				{"navigation", "route", "info", "Route matched: POST /checkout", nil},
				{"default", "db.sql.query", "info", "select * from `carts` where `id` = ? limit 1", map[string]any{"executionTimeMs": 1.8, "connectionName": "mysql"}},
				{"default", "cache", "info", "Read: cart:c7f1e2:totals", nil},
				{"default", "db.sql.transaction", "info", "BEGIN", nil},
				{"default", "db.sql.query", "info", "insert into `orders` (`number`, `customer_id`, `total`, `currency`) values (?, ?, ?, ?)", map[string]any{"executionTimeMs": 4.2, "connectionName": "mysql"}},
			},
			times: 4,
		},
		{
			excType: "App\\Exceptions\\OutOfStockException",
			value:   "Product SKU-1042 \"Trail Runner X\" (size 42) is out of stock: requested 2, available 0",
			level:   "error", handled: true, method: "POST", path: "/cart/items", route: "POST /cart/items",
			body: map[string]any{"sku": "SKU-1042", "size": "42", "quantity": 2},
			frames: []frame{
				{file: "/var/www/html/app/Inventory/StockGuard.php", fn: "App\\Inventory\\StockGuard::reserve", line: 27, inApp: true,
					pre:  []string{"        $available = $this->stock->available($sku, $variant);", "", "        if ($available < $quantity) {"},
					code: "            throw new OutOfStockException($sku, $variant, $quantity, $available);",
					post: []string{"        }", "", "        $this->stock->reserve($sku, $variant, $quantity);"}},
				{file: "/var/www/html/app/Http/Controllers/CartController.php", fn: "App\\Http\\Controllers\\CartController::add", line: 33, inApp: true,
					pre:  []string{"    public function add(AddToCartRequest $request): JsonResponse", "    {"},
					code: "        $this->stock->reserve($request->sku(), $request->variant(), $request->quantity());",
					post: []string{"        $item = $this->cart->add($request->sku(), $request->variant(), $request->quantity());"}},
			},
			tags: map[string]string{"sku": "SKU-1042", "warehouse": "berlin-1"},
			crumbs: []crumb{
				{"navigation", "route", "info", "Route matched: POST /cart/items", nil},
				{"default", "db.sql.query", "info", "select `available` from `stock_levels` where `sku` = ? and `variant` = ?", map[string]any{"executionTimeMs": 0.9, "connectionName": "mysql"}},
			},
			times: 3,
		},
		{
			excType: "ErrorException",
			value:   "Undefined array key \"discount_code\"",
			level:   "warning", handled: true, method: "GET", path: "/cart", route: "GET /cart",
			frames: []frame{
				{file: "/var/www/html/app/Pricing/PromotionResolver.php", fn: "App\\Pricing\\PromotionResolver::forSession", line: 19, inApp: true,
					pre:  []string{"    public function forSession(array $session): ?Promotion", "    {"},
					code: "        $code = $session['discount_code'];",
					post: []string{"", "        return $code ? $this->promotions->findActive($code) : null;"}},
				{file: "/var/www/html/app/Http/Controllers/CartController.php", fn: "App\\Http\\Controllers\\CartController::show", line: 21, inApp: true,
					code: "        $promotion = $this->promotions->forSession($request->session()->all());"},
			},
			crumbs: []crumb{{"navigation", "route", "info", "Route matched: GET /cart", nil}},
			times:  2,
		},
	}
}

func billingErrors() []errorSpec {
	return []errorSpec{
		{
			excType: "App\\Billing\\Exceptions\\PaymentDeclinedException",
			value:   "Card declined: insufficient_funds (invoice INV-2026-0042)",
			level:   "error", handled: true, method: "POST", path: "/api/v1/charges", route: "POST /api/v1/charges",
			body: map[string]any{"invoice": "INV-2026-0042", "amount": 4900, "currency": "EUR", "payment_method": "pm_card_visa_debit"},
			frames: []frame{
				{file: "/var/www/html/app/Billing/Gateway/CardGateway.php", fn: "App\\Billing\\Gateway\\CardGateway::charge", line: 74, inApp: true,
					pre:  []string{"        $response = $this->http->post('/v1/charges', $payload);", "", "        if ($response->json('status') === 'declined') {"},
					code: "            throw PaymentDeclinedException::fromResponse($response, $invoice);",
					post: []string{"        }", "", "        return Charge::fromResponse($response);"}},
				{file: "/var/www/html/app/Billing/ChargeService.php", fn: "App\\Billing\\ChargeService::chargeInvoice", line: 46, inApp: true,
					code: "        $charge = $this->gateway->charge($invoice, $invoice->customer->defaultPaymentMethod());"},
				{file: "/var/www/html/app/Http/Controllers/Api/ChargeController.php", fn: "App\\Http\\Controllers\\Api\\ChargeController::store", line: 29, inApp: true,
					code: "        $charge = $this->charges->chargeInvoice($invoice);"},
			},
			tags:  map[string]string{"gateway": "card", "decline_code": "insufficient_funds"},
			extra: map[string]any{"invoice_id": "INV-2026-0042", "attempt": 2},
			crumbs: []crumb{
				{"navigation", "route", "info", "Route matched: POST /api/v1/charges", nil},
				{"default", "db.sql.query", "info", "select * from `invoices` where `number` = ? limit 1", map[string]any{"executionTimeMs": 1.1, "connectionName": "pgsql"}},
				{"http", "http", "info", "POST https://payments.example.net/v1/charges [402]", map[string]any{"method": "POST", "status_code": 402, "url": "https://payments.example.net/v1/charges"}},
			},
			times: 5,
		},
		{
			excType: "GuzzleHttp\\Exception\\ConnectException",
			value:   "cURL error 28: Operation timed out after 10001 milliseconds with 0 bytes received for https://tax.example.net/v2/calculate",
			level:   "error", method: "POST", path: "/api/v1/invoices", route: "POST /api/v1/invoices",
			body: map[string]any{"customer": "acct_7781", "lines": []map[string]any{{"sku": "plan-pro-monthly", "qty": 1}}},
			frames: []frame{
				vendor("guzzlehttp/guzzle/src/Handler/CurlFactory.php", "GuzzleHttp\\Handler\\CurlFactory::createRejection", 276),
				vendor("guzzlehttp/guzzle/src/Client.php", "GuzzleHttp\\Client::request", 189),
				{file: "/var/www/html/app/Billing/Tax/TaxClient.php", fn: "App\\Billing\\Tax\\TaxClient::calculate", line: 38, inApp: true,
					pre:  []string{"    public function calculate(Invoice $invoice): TaxBreakdown", "    {"},
					code: "        $response = $this->client->post('/v2/calculate', ['json' => $this->payload($invoice), 'timeout' => 10]);",
					post: []string{"", "        return TaxBreakdown::fromArray($response->json());"}},
				{file: "/var/www/html/app/Http/Controllers/Api/InvoiceController.php", fn: "App\\Http\\Controllers\\Api\\InvoiceController::store", line: 52, inApp: true,
					code: "        $invoice->applyTax($this->tax->calculate($invoice));"},
			},
			tags: map[string]string{"upstream": "tax"},
			crumbs: []crumb{
				{"navigation", "route", "info", "Route matched: POST /api/v1/invoices", nil},
				{"http", "http", "error", "POST https://tax.example.net/v2/calculate [timeout]", map[string]any{"method": "POST", "url": "https://tax.example.net/v2/calculate"}},
			},
			times: 2,
		},
		{
			excType: "App\\Billing\\Exceptions\\InvalidWebhookSignature",
			value:   "Webhook signature verification failed: timestamp outside the 300s tolerance",
			level:   "warning", handled: true, method: "POST", path: "/webhooks/payments", route: "POST /webhooks/payments",
			frames: []frame{
				{file: "/var/www/html/app/Http/Middleware/VerifyWebhookSignature.php", fn: "App\\Http\\Middleware\\VerifyWebhookSignature::handle", line: 31, inApp: true,
					pre:  []string{"        if (abs(time() - $timestamp) > self::TOLERANCE) {"},
					code: "            throw new InvalidWebhookSignature('timestamp outside the 300s tolerance');",
					post: []string{"        }"}},
			},
			crumbs: []crumb{{"navigation", "route", "info", "Route matched: POST /webhooks/payments", nil}},
			times:  1,
		},
	}
}

// --- transactions --------------------------------------------------------------

type spanSpec struct {
	op, desc   string
	start, dur float64 // ms from the transaction start
	parent     int     // index of the parent span, -1 = transaction root
	data       map[string]any
	failRate   int // percent of runs where the span fails
}

type txnSpec struct {
	name, op string
	method   string
	path     string
	dur      float64 // ms
	spans    []spanSpec
}

func mysql(desc string, start, dur float64) spanSpec {
	return spanSpec{op: "db.sql.query", desc: desc, start: start, dur: dur, parent: -1,
		data: map[string]any{"db.system": "mysql", "db.name": "shop", "server.address": "mysql.shop.internal", "server.port": 3306}}
}

func pgsql(desc string, start, dur float64) spanSpec {
	return spanSpec{op: "db.sql.query", desc: desc, start: start, dur: dur, parent: -1,
		data: map[string]any{"db.system": "postgresql", "db.name": "billing", "server.address": "postgres.billing.internal", "server.port": 5432}}
}

func cacheSpan(op, key, server string, start, dur float64, hit bool) spanSpec {
	return spanSpec{op: op, desc: key, start: start, dur: dur, parent: -1,
		data: map[string]any{"cache.key": []string{key}, "cache.hit": hit, "server.address": server, "network.peer.port": 6379}}
}

func httpClient(method, url string, start, dur float64, status, failRate int) spanSpec {
	return spanSpec{op: "http.client", desc: method + " " + url, start: start, dur: dur, parent: -1, failRate: failRate,
		data: map[string]any{"http.url": url, "http.request.method": method, "http.response.status_code": status}}
}

func shopTxns() []txnSpec {
	return []txnSpec{
		{name: "GET /products/{slug}", op: "http.server", method: "GET", path: "/products/trail-runner-x", dur: 96, spans: []spanSpec{
			{op: "middleware.handle", desc: "App\\Http\\Middleware\\Authenticate", start: 2, dur: 6, parent: -1},
			cacheSpan("cache.get", "product:trail-runner-x", "redis.shop.internal", 9, 1.2, false),
			mysql("select * from `products` where `slug` = ? and `published` = 1 limit 1", 11, 8.4),
			mysql("select * from `product_images` where `product_id` in (?)", 20, 3.1),
			cacheSpan("cache.put", "product:trail-runner-x", "redis.shop.internal", 24, 0.8, false),
			{op: "view.render", desc: "products.show", start: 30, dur: 58, parent: -1},
		}},
		{name: "GET /cart", op: "http.server", method: "GET", path: "/cart", dur: 54, spans: []spanSpec{
			cacheSpan("cache.get", "cart:c7f1e2:totals", "redis.shop.internal", 3, 0.9, true),
			mysql("select * from `cart_items` where `cart_id` = ?", 6, 4.6),
			{op: "view.render", desc: "cart.index", start: 14, dur: 34, parent: -1},
		}},
		{name: "POST /checkout", op: "http.server", method: "POST", path: "/checkout", dur: 420, spans: []spanSpec{
			mysql("select * from `carts` where `id` = ? limit 1", 4, 1.9),
			{op: "db.sql.transaction", desc: "BEGIN", start: 8, dur: 0.4, parent: -1,
				data: map[string]any{"db.system": "mysql", "server.address": "mysql.shop.internal"}},
			mysql("insert into `orders` (`number`, `customer_id`, `total`, `currency`) values (?, ?, ?, ?)", 9, 5.2),
			mysql("insert into `order_items` (`order_id`, `sku`, `quantity`, `price`) values (?, ?, ?, ?), (?, ?, ?, ?)", 15, 3.8),
			httpClient("POST", "http://billing-api/api/v1/charges", 22, 310, 201, 10),
			{op: "queue.publish", desc: "App\\Jobs\\SendOrderConfirmation", start: 336, dur: 2.6, parent: -1,
				data: map[string]any{"messaging.system": "redis", "messaging.destination": "emails"}},
			cacheSpan("cache.forget", "cart:c7f1e2:totals", "redis.shop.internal", 340, 0.7, false),
		}},
		{name: "GET /search", op: "http.server", method: "GET", path: "/search?q=rain+jacket", dur: 140, spans: []spanSpec{
			httpClient("POST", "http://search.shop.internal:9200/products/_search", 5, 88, 200, 0),
			mysql("select * from `products` where `id` in (?, ?, ?, ?, ?, ?, ?, ?)", 96, 6.3),
			{op: "view.render", desc: "search.results", start: 105, dur: 30, parent: -1},
		}},
	}
}

func billingTxns() []txnSpec {
	return []txnSpec{
		{name: "GET /api/v1/invoices/{invoice}", op: "http.server", method: "GET", path: "/api/v1/invoices/INV-2026-0042", dur: 38, spans: []spanSpec{
			cacheSpan("cache.get", "invoice:INV-2026-0042", "redis.billing.internal", 2, 0.8, false),
			pgsql("select * from invoices where number = $1 limit 1", 4, 3.2),
			pgsql("select * from invoice_lines where invoice_id = $1", 8, 2.4),
		}},
		{name: "POST /api/v1/invoices", op: "http.server", method: "POST", path: "/api/v1/invoices", dur: 210, spans: []spanSpec{
			pgsql("select * from customers where id = $1", 3, 1.6),
			httpClient("POST", "https://tax.example.net/v2/calculate", 7, 160, 200, 15),
			pgsql("insert into invoices (number, customer_id, total, currency, due_at) values ($1, $2, $3, $4, $5)", 170, 4.1),
			{op: "queue.publish", desc: "App\\Jobs\\GenerateInvoicePdf", start: 178, dur: 1.9, parent: -1,
				data: map[string]any{"messaging.system": "redis", "messaging.destination": "invoices"}},
		}},
		{name: "App\\Jobs\\GenerateInvoicePdf", op: "queue.process", dur: 640, spans: []spanSpec{
			pgsql("select * from invoices where id = $1", 3, 2.2),
			{op: "view.render", desc: "pdf.invoice", start: 8, dur: 410, parent: -1},
			{op: "file.write", desc: "invoices/2026/10/INV-2026-0042.pdf", start: 425, dur: 180, parent: -1},
		}},
	}
}

// chargeTxn is billing's side of shop's checkout call: it continues shop's trace.
var chargeTxn = txnSpec{name: "POST /api/v1/charges", op: "http.server", method: "POST", path: "/api/v1/charges", dur: 290, spans: []spanSpec{
	pgsql("select * from invoices where number = $1 limit 1", 3, 1.4),
	httpClient("POST", "https://payments.example.net/v1/charges", 8, 240, 200, 8),
	pgsql("update invoices set status = $1, paid_at = $2 where id = $3", 252, 2.9),
	cacheSpan("cache.forget", "invoice:INV-2026-0042", "redis.billing.internal", 257, 0.6, false),
}}

type builtTxn struct {
	payload  map[string]any
	traceID  string
	spanIDs  []string
	startTS  time.Time
	duration float64 // ms
	failed   bool
}

// buildTxn renders a transaction starting at the given time. Durations get a
// random 0.7x–1.6x jitter so charts and percentiles look alive.
func buildTxn(a app, t txnSpec, start time.Time, traceID, parentSpanID string) builtTxn {
	scale := float64(rnd(70, 160)) / 100
	at := func(ms float64) time.Time { return start.Add(time.Duration(ms * scale * float64(time.Millisecond))) }

	rootSpan := randHex(8)
	ids := make([]string, len(t.spans))
	for i := range ids {
		ids[i] = randHex(8)
	}

	failed := false
	spans := make([]map[string]any, len(t.spans))
	for i, sp := range t.spans {
		parent := rootSpan
		if sp.parent >= 0 {
			parent = ids[sp.parent]
		}
		status := "ok"
		data := map[string]any{"service.name": a.service}
		for k, v := range sp.data {
			data[k] = v
		}
		if sp.failRate > 0 && rnd(1, 100) <= sp.failRate {
			status = "internal_error"
			failed = true
			if _, ok := data["http.response.status_code"]; ok {
				data["http.response.status_code"] = 503
			}
		}
		spans[i] = map[string]any{
			"span_id":         ids[i],
			"parent_span_id":  parent,
			"trace_id":        traceID,
			"op":              sp.op,
			"description":     sp.desc,
			"status":          status,
			"start_timestamp": epoch(at(sp.start)),
			"timestamp":       epoch(at(sp.start + sp.dur)),
			"data":            data,
			"origin":          "auto",
		}
	}

	status := "ok"
	if failed {
		status = "internal_error"
	}
	trace := map[string]any{
		"trace_id": traceID, "span_id": rootSpan, "op": t.op, "status": status, "origin": "auto.http.server",
		"data": map[string]any{"service.name": a.service, "http.request.method": t.method},
	}
	if parentSpanID != "" {
		trace["parent_span_id"] = parentSpanID
	}

	p := map[string]any{
		"event_id":         randHex(16),
		"type":             "transaction",
		"transaction":      t.name,
		"transaction_info": map[string]any{"source": "route"},
		"op":               t.op,
		"status":           status,
		"start_timestamp":  epoch(start),
		"timestamp":        epoch(at(t.dur)),
		"platform":         "php",
		"server_name":      a.server,
		"release":          a.release,
		"environment":      "production",
		"sdk":              sentrySDK,
		"contexts": map[string]any{
			"trace":   trace,
			"runtime": map[string]any{"name": "php", "version": "8.3.12"},
		},
		"spans": spans,
	}
	if t.method != "" {
		p["request"] = map[string]any{"url": "https://" + a.host + t.path, "method": t.method}
	}
	return builtTxn{payload: p, traceID: traceID, spanIDs: ids, startTS: start, duration: t.dur * scale, failed: failed}
}

// --- logs and metrics -------------------------------------------------------------

var severity = map[string]int{"trace": 1, "debug": 5, "info": 9, "warn": 13, "error": 17, "fatal": 21}

func attr(v any) map[string]any {
	switch v.(type) {
	case int:
		return map[string]any{"type": "integer", "value": v}
	case float64:
		return map[string]any{"type": "double", "value": v}
	case bool:
		return map[string]any{"type": "boolean", "value": v}
	}
	return map[string]any{"type": "string", "value": fmt.Sprint(v)}
}

func (a app) logRecord(at time.Time, level, body, traceID string, attrs map[string]any) map[string]any {
	all := map[string]any{
		"sentry.release":     attr(a.release),
		"sentry.environment": attr("production"),
		"sentry.sdk.name":    attr("sentry.php.laravel"),
		"server.address":     attr(a.server),
	}
	for k, v := range attrs {
		all[k] = attr(v)
	}
	rec := map[string]any{
		"timestamp":       epoch(at),
		"level":           level,
		"severity_number": severity[level],
		"body":            body,
		"attributes":      all,
	}
	if traceID != "" {
		rec["trace_id"] = traceID
	}
	return rec
}

func metric(at time.Time, name, typ string, value float64, unit, traceID string, attrs map[string]any) map[string]any {
	m := map[string]any{"timestamp": epoch(at), "name": name, "type": typ, "value": value}
	if unit != "" {
		m["unit"] = unit
	}
	if traceID != "" {
		m["trace_id"] = traceID
		m["span_id"] = randHex(8)
	}
	if attrs != nil {
		typed := map[string]any{}
		for k, v := range attrs {
			typed[k] = attr(v)
		}
		m["attributes"] = typed
	}
	return m
}

// --- seeding ---------------------------------------------------------------------

func (s *seeder) seedSentry() {
	logs := map[string][]map[string]any{}
	metrics := map[string][]map[string]any{}

	// Errors, each repeated so grouping shows counts.
	for _, set := range []struct {
		a      app
		errors []errorSpec
	}{{shopApp, shopErrors()}, {billingApp, billingErrors()}} {
		for _, e := range set.errors {
			for i := 0; i < e.times; i++ {
				at := s.ago(rnd(1, 50))
				trace := randHex(16)
				if err := s.sendError(set.a, e, at, trace); err != nil {
					s.fail("sentry error "+e.excType, err)
					continue
				}
				s.count("sentry errors", 1)
				lvl := "error"
				if e.level == "warning" {
					lvl = "warn"
				}
				logs[set.a.project] = append(logs[set.a.project], set.a.logRecord(at, lvl, e.excType+": "+truncate(e.value, 120), trace,
					map[string]any{"exception.type": e.excType, "http.route": e.route}))
			}
		}
	}

	// Transactions spread over the last ~55 minutes.
	sendTxn := func(a app, b builtTxn) {
		if err := s.sendEnvelope(a.project, b.payload["event_id"].(string), envItem{typ: "transaction", payload: b.payload,
			extra: map[string]any{"content_type": "application/json"}}); err != nil {
			s.fail("sentry transaction "+b.payload["transaction"].(string), err)
			return
		}
		s.count("sentry transactions", 1)
		s.count("sentry spans", len(b.spanIDs))
		route := b.payload["transaction"].(string)
		metrics[a.project] = append(metrics[a.project],
			metric(b.startTS, "http.server.duration", "distribution", round1(b.duration), "millisecond", b.traceID, map[string]any{"http.route": route}))
	}

	for _, t := range shopTxns() {
		for i, n := 0, rnd(4, 7); i < n; i++ {
			start := s.ago(rnd(1, 55))
			b := buildTxn(shopApp, t, start, randHex(16), "")
			sendTxn(shopApp, b)
			at := start.Add(time.Duration(b.duration) * time.Millisecond)

			if t.name == "POST /checkout" {
				order := fmt.Sprintf("ORD-2026-11-%05d", rnd(10000, 99999))
				total := float64(rnd(1990, 34900)) / 100
				// Billing continues the trace from the shop's http.client span.
				charge := buildTxn(billingApp, chargeTxn, start.Add(25*time.Millisecond), b.traceID, b.spanIDs[4])
				sendTxn(billingApp, charge)
				logs["shop"] = append(logs["shop"], shopApp.logRecord(at, "info", "Order "+order+" placed", b.traceID,
					map[string]any{"order.id": order, "order.total": total, "order.currency": "EUR", "order.items": rnd(1, 5)}))
				logs["billing"] = append(logs["billing"], billingApp.logRecord(charge.startTS, "info", fmt.Sprintf("Charged %.2f EUR for order %s", total, order), b.traceID,
					map[string]any{"order.id": order, "amount": total}))
				metrics["shop"] = append(metrics["shop"],
					metric(at, "orders.placed", "counter", 1, "", b.traceID, map[string]any{"payment.method": pick([]string{"card", "paypal", "invoice"})}),
					metric(at, "order.value", "distribution", total, "", b.traceID, map[string]any{"currency": "EUR"}))
				if b.failed || charge.failed {
					logs["shop"] = append(logs["shop"], shopApp.logRecord(at, "error", "Payment call to billing failed for "+order+", order kept in pending_payment", b.traceID,
						map[string]any{"order.id": order, "upstream": "billing-api"}))
				}
			}
			if t.name == "GET /search" {
				logs["shop"] = append(logs["shop"], shopApp.logRecord(at, "debug", "Search \"rain jacket\" returned 8 hits", b.traceID,
					map[string]any{"search.query": "rain jacket", "search.hits": 8}))
			}
		}
	}
	for _, t := range billingTxns() {
		for i, n := 0, rnd(3, 6); i < n; i++ {
			start := s.ago(rnd(1, 55))
			b := buildTxn(billingApp, t, start, randHex(16), "")
			sendTxn(billingApp, b)
			if t.op == "queue.process" {
				logs["billing"] = append(logs["billing"], billingApp.logRecord(start, "info", "Invoice PDF generated", b.traceID,
					map[string]any{"invoice.id": "INV-2026-0042", "pdf.size_kb": rnd(38, 64)}))
			}
		}
	}

	// Standalone logs and gauges, one point every few minutes.
	for m := 55; m >= 0; m -= 5 {
		at := s.ago(m)
		metrics["shop"] = append(metrics["shop"],
			metric(at, "queue.emails.depth", "gauge", float64(rnd(0, 40)), "", "", map[string]any{"queue": "emails"}),
			metric(at, "cache.hit_ratio", "gauge", float64(rnd(78, 97))/100, "ratio", "", nil))
		metrics["billing"] = append(metrics["billing"],
			metric(at, "subscriptions.active", "gauge", float64(1280+rnd(0, 25)), "", "", nil),
			metric(at, "charges.declined", "counter", float64(rnd(0, 3)), "", "", map[string]any{"gateway": "card"}))
	}
	logs["shop"] = append(logs["shop"],
		shopApp.logRecord(s.ago(52), "info", "Deployment shop@2.14.0 finished, 3 workers restarted", "", map[string]any{"deploy.id": "d-7731"}),
		shopApp.logRecord(s.ago(31), "warn", "Slow query: select * from `products` where `category_id` = ? took 812 ms", "", map[string]any{"db.duration_ms": 812}),
		shopApp.logRecord(s.ago(12), "trace", "Feature flag new-checkout-flow evaluated: on", "", map[string]any{"flag": "new-checkout-flow"}),
	)
	logs["billing"] = append(logs["billing"],
		billingApp.logRecord(s.ago(48), "info", "Nightly dunning run: 14 reminders queued", "", map[string]any{"job": "billing:send-reminders", "count": 14}),
		billingApp.logRecord(s.ago(22), "warn", "Tax provider answered in 7.8 s, close to the 10 s timeout", "", map[string]any{"upstream": "tax", "duration_ms": 7800}),
		billingApp.logRecord(s.ago(4), "fatal", "Ledger balance mismatch for account acct_3345: expected 120.00, got 118.50", "", map[string]any{"account": "acct_3345"}),
	)

	for _, a := range []app{shopApp, billingApp} {
		l, m := logs[a.project], metrics[a.project]
		err := s.sendEnvelope(a.project, "",
			envItem{typ: "log", payload: map[string]any{"items": l}, extra: map[string]any{"item_count": len(l), "content_type": "application/vnd.sentry.items.log+json"}},
			envItem{typ: "trace_metric", payload: map[string]any{"items": m}, extra: map[string]any{"item_count": len(m), "content_type": "application/vnd.sentry.items.trace-metric+json"}},
		)
		if err != nil {
			s.fail("sentry logs/metrics "+a.project, err)
			continue
		}
		s.count("sentry logs", len(l))
		s.count("sentry metrics", len(m))
	}
}

func round1(f float64) float64 { return float64(int(f*10)) / 10 }
