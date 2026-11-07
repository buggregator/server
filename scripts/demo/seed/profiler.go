package main

import (
	"encoding/json"
)

// pnode is one function in an XHProf call tree. Self costs are per all calls.
type pnode struct {
	fn   string
	ct   int64
	wt   int64 // self wall time, µs
	mu   int64 // self memory, bytes
	io   bool  // waits on I/O: little CPU
	kids []*pnode
}

func fn(name string, ct int64, wtUs int64, kids ...*pnode) *pnode {
	return &pnode{fn: name, ct: ct, wt: wtUs, mu: wtUs * 24, kids: kids}
}

func ioFn(name string, ct int64, wtUs int64, kids ...*pnode) *pnode {
	n := fn(name, ct, wtUs, kids...)
	n.io = true
	n.mu = 4096 * ct
	return n
}

func mem(n *pnode, bytes int64) *pnode { n.mu = bytes; return n }

type xhMetrics struct {
	Wt  int64 `json:"wt"`
	CPU int64 `json:"cpu"`
	Mu  int64 `json:"mu"`
	Pmu int64 `json:"pmu"`
	Ct  int64 `json:"ct"`
}

// flatten turns the tree into XHProf "parent==>child" edges with inclusive
// costs. jitter (percent) varies the wall time per run.
func flatten(root *pnode, jitter int) map[string]xhMetrics {
	out := map[string]xhMetrics{}
	var walk func(n *pnode, parent string) xhMetrics
	walk = func(n *pnode, parent string) xhMetrics {
		f := float64(100+rnd(-jitter, jitter)) / 100
		wt := int64(float64(n.wt) * f)
		cpu := wt * 85 / 100
		if n.io {
			cpu = wt / 12
		}
		m := xhMetrics{Wt: wt, CPU: cpu, Mu: n.mu, Ct: n.ct}
		for _, k := range n.kids {
			c := walk(k, n.fn)
			m.Wt += c.Wt
			m.CPU += c.CPU
			m.Mu += c.Mu
		}
		m.Pmu = m.Mu + m.Mu/4
		key := n.fn
		if parent != "" {
			key = parent + "==>" + n.fn
		}
		if prev, ok := out[key]; ok {
			prev.Wt += m.Wt
			prev.CPU += m.CPU
			prev.Mu += m.Mu
			prev.Pmu += m.Pmu
			prev.Ct += m.Ct
			m = prev
		}
		out[key] = m
		return m
	}
	walk(root, "")
	return out
}

func laravelBoot() []*pnode {
	return []*pnode{
		mem(fn("Composer\\Autoload\\ClassLoader::loadClass", 412, 9000), 3<<20),
		fn("Illuminate\\Foundation\\Application::bootstrapWith", 1, 1200,
			fn("Illuminate\\Foundation\\Bootstrap\\LoadEnvironmentVariables::bootstrap", 1, 700),
			fn("Illuminate\\Foundation\\Bootstrap\\LoadConfiguration::bootstrap", 1, 3400),
			fn("Illuminate\\Foundation\\Bootstrap\\RegisterProviders::bootstrap", 1, 2100,
				fn("Illuminate\\Foundation\\Application::register", 38, 4800)),
			fn("Illuminate\\Foundation\\Bootstrap\\BootProviders::bootstrap", 1, 900,
				fn("App\\Providers\\AppServiceProvider::boot", 1, 600),
				fn("App\\Providers\\RouteServiceProvider::boot", 1, 1300)),
		),
	}
}

func shopCatalogProfile() *pnode {
	kids := laravelBoot()
	kids = append(kids,
		fn("Illuminate\\Foundation\\Http\\Kernel::handle", 1, 400,
			fn("Illuminate\\Routing\\Router::dispatch", 1, 300,
				fn("Illuminate\\Pipeline\\Pipeline::then", 1, 800,
					fn("App\\Http\\Middleware\\Authenticate::handle", 1, 500,
						fn("Illuminate\\Session\\Store::start", 1, 300,
							ioFn("Redis::get", 1, 900))),
					fn("App\\Http\\Controllers\\CatalogController::index", 1, 700,
						fn("App\\Repositories\\ProductRepository::paginate", 1, 600,
							fn("Illuminate\\Database\\Connection::select", 3, 900,
								ioFn("PDOStatement::execute", 3, 18400)),
							mem(fn("Illuminate\\Database\\Eloquent\\Builder::hydrate", 2, 6200,
								fn("Illuminate\\Database\\Eloquent\\Model::newFromBuilder", 48, 3100)), 2<<20)),
						fn("App\\Services\\PriceCalculator::forProducts", 1, 1200,
							fn("App\\Services\\CurrencyConverter::convert", 48, 4300,
								fn("Illuminate\\Cache\\Repository::remember", 48, 2500,
									ioFn("Redis::get", 48, 7200))),
							fn("App\\Services\\TaxRules::rateFor", 48, 1900)),
						fn("Illuminate\\View\\View::render", 1, 900,
							fn("Illuminate\\View\\Engines\\CompilerEngine::get", 14, 1100,
								mem(fn("Illuminate\\View\\Engines\\PhpEngine::evaluatePath", 14, 21000), 1536<<10))),
					),
				),
			),
		),
		fn("Symfony\\Component\\HttpFoundation\\Response::send", 1, 1400),
		fn("Illuminate\\Foundation\\Http\\Kernel::terminate", 1, 200,
			fn("Illuminate\\Session\\Middleware\\StartSession::terminate", 1, 300,
				ioFn("Redis::setex", 1, 1100))),
	)
	return fn("main()", 1, 300, kids...)
}

func billingChargeProfile() *pnode {
	kids := laravelBoot()
	kids = append(kids,
		fn("Illuminate\\Foundation\\Http\\Kernel::handle", 1, 400,
			fn("Illuminate\\Routing\\Router::dispatch", 1, 300,
				fn("Illuminate\\Pipeline\\Pipeline::then", 1, 700,
					fn("App\\Http\\Middleware\\VerifyApiToken::handle", 1, 400,
						fn("Illuminate\\Database\\Connection::select", 1, 300,
							ioFn("PDOStatement::execute", 1, 2100))),
					fn("App\\Http\\Controllers\\Api\\ChargeController::store", 1, 600,
						fn("App\\Billing\\ChargeService::chargeInvoice", 1, 900,
							fn("Illuminate\\Database\\Connection::select", 2, 600,
								ioFn("PDOStatement::execute", 2, 5200)),
							fn("App\\Billing\\Gateway\\CardGateway::charge", 1, 800,
								fn("GuzzleHttp\\Client::request", 1, 1500,
									ioFn("curl_exec", 1, 212000)),
								fn("json_decode", 1, 400)),
							fn("Illuminate\\Database\\Connection::update", 1, 400,
								ioFn("PDOStatement::execute", 1, 3300)),
							fn("Illuminate\\Events\\Dispatcher::dispatch", 1, 500,
								fn("App\\Listeners\\SendReceiptEmail::handle", 1, 700,
									fn("Illuminate\\Mail\\Mailer::send", 1, 1200,
										mem(fn("Illuminate\\View\\Engines\\PhpEngine::evaluatePath", 3, 6400), 768<<10),
										ioFn("Symfony\\Component\\Mailer\\Transport\\Smtp\\SmtpTransport::send", 1, 38000)))),
						),
						fn("Illuminate\\Http\\JsonResponse::setData", 1, 900,
							fn("json_encode", 1, 600)),
					),
				),
			),
		),
		fn("Symfony\\Component\\HttpFoundation\\Response::send", 1, 700),
	)
	return fn("main()", 1, 250, kids...)
}

func (s *seeder) seedProfiler() {
	for _, p := range []struct {
		project, app, host string
		tree               func() *pnode
		tags               map[string]any
	}{
		{"shop", "shop", "shop-web-1", shopCatalogProfile, map[string]any{"method": "GET", "url": "/catalog/running?page=2", "route": "catalog.index", "env": "production", "php": "8.3.12"}},
		{"shop", "shop", "shop-web-2", shopCatalogProfile, map[string]any{"method": "GET", "url": "/catalog/jackets", "route": "catalog.index", "env": "production", "php": "8.3.12"}},
		{"billing", "billing", "billing-api-1", billingChargeProfile, map[string]any{"method": "POST", "url": "/api/v1/charges", "route": "api.charges.store", "env": "production", "php": "8.3.12"}},
		{"billing", "billing", "billing-api-1", billingChargeProfile, map[string]any{"method": "POST", "url": "/api/v1/charges", "route": "api.charges.store", "env": "production", "php": "8.3.12", "retry": "true"}},
	} {
		body, _ := json.Marshal(map[string]any{
			"profile":  flatten(p.tree(), 25),
			"tags":     p.tags,
			"app_name": p.app,
			"hostname": p.host,
			"date":     s.ago(rnd(1, 30)).Unix(),
		})
		h := projectHeaders("profiler", p.project)
		h["Content-Type"] = "application/json"
		if err := s.post("POST", "/api/profiler/store", h, body); err != nil {
			s.fail("profiler "+p.tags["url"].(string), err)
			continue
		}
		s.count("profiler", 1)
	}
}
