package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/buggregator/go-buggregator/internal/app"
	mcpserver "github.com/buggregator/go-buggregator/internal/mcp"
	"github.com/buggregator/go-buggregator/internal/metrics"
	"github.com/buggregator/go-buggregator/internal/module"
	httpserver "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/buggregator/go-buggregator/modules/httpdumps"
	"github.com/buggregator/go-buggregator/modules/inspector"
	"github.com/buggregator/go-buggregator/modules/monolog"
	"github.com/buggregator/go-buggregator/modules/profiler"
	"github.com/buggregator/go-buggregator/modules/proxy"
	"github.com/buggregator/go-buggregator/modules/ray"
	"github.com/buggregator/go-buggregator/modules/sentry"
	"github.com/buggregator/go-buggregator/modules/sms"
	smtpmod "github.com/buggregator/go-buggregator/modules/smtp"
	"github.com/buggregator/go-buggregator/modules/vardumper"
	"github.com/buggregator/go-buggregator/modules/webhooks"
	"github.com/buggregator/go-buggregator/modules/xdebug"
	"github.com/buggregator/go-buggregator/modules/xdebugtrace"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// MCP proxy subcommand: "buggregator mcp" bridges stdio to the running instance.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		socketPath := os.Getenv("MCP_SOCKET_PATH")
		if socketPath == "" {
			socketPath = "/tmp/buggregator-mcp.sock"
		}
		if err := mcpserver.RunProxy(socketPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := app.LoadConfig()
	cfg.Version = version

	if limit, source := app.ApplyMemoryLimit(); limit > 0 {
		slog.Info("go memory limit", "mib", limit>>20, "source", source)
	}

	db, err := storage.OpenPooled(cfg.DatabaseDSN, cfg.Database.MaxOpenConns)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Seed projects from config file.
	for _, p := range cfg.Projects {
		db.Exec(`INSERT OR IGNORE INTO projects (key, name) VALUES (?, ?)`, p.Key, p.Name)
	}

	// Create metrics collector if enabled.
	var collector *metrics.Collector
	if cfg.Metrics.Enabled {
		collector = metrics.NewCollector()
		slog.Info("prometheus metrics enabled")
	}

	store := storage.NewSQLiteStore(db)
	attachments := storage.NewAttachmentStore(cfg.Storage.Mode, cfg.Storage.Path)
	hub := ws.NewHub()
	registry := module.NewRegistry()
	enabled := cfg.Modules

	// TCP modules.
	monologMod := monolog.New(cfg.MonologAddr)
	smtpMod := smtpmod.New(cfg.SMTPAddr, attachments, db)
	vardumperMod := vardumper.New(cfg.VarDumperAddr)
	proxyMod := proxy.New(cfg.ProxyAddr)

	// Start VarDumper PHP parser (only if enabled).
	//
	// VarDumper bundles a standalone PHP runtime via static-php's micro.sfx,
	// which has no Windows build. The embedded binary on Windows is a
	// placeholder stub, so the parser would crash immediately — skip start-up
	// entirely there and let the rest of the server run. TCPServers() returns
	// nil when the parser isn't running, so the :9912 listener is also skipped.
	if enabled.IsEnabled("var-dump") {
		if runtime.GOOS == "windows" {
			slog.Warn("VarDumper module is not supported on Windows; module disabled")
		} else {
			if err := vardumperMod.StartPHP(); err != nil {
				slog.Error("failed to start VarDumper PHP parser", "err", err)
				os.Exit(1)
			}
			defer vardumperMod.StopPHP()
		}
	}

	// Register enabled modules.
	if enabled.IsEnabled("sentry") {
		registry.Register(sentry.New())
	}
	if enabled.IsEnabled("ray") {
		registry.Register(ray.New())
	}
	if enabled.IsEnabled("inspector") {
		registry.Register(inspector.New())
	}
	if enabled.IsEnabled("profiler") {
		registry.Register(profiler.New())
	}
	if enabled.IsEnabled("xdebug-trace") {
		// Trace files live on disk either way: under storage.path in
		// filesystem mode, in a temp dir removed at exit in memory mode.
		// The module is optional: without a dir it stays off, the rest runs.
		traceDir := filepath.Join(cfg.Storage.Path, "xdebug-traces")
		if cfg.Storage.Mode != "filesystem" {
			tmp, err := os.MkdirTemp("", "buggregator-xdebug-traces-")
			if err != nil {
				slog.Error("xdebug-trace: cannot create temp dir; module disabled", "err", err)
				traceDir = ""
			} else {
				defer os.RemoveAll(tmp)
				traceDir = tmp
			}
		}
		if traceDir != "" {
			registry.Register(xdebugtrace.New(traceDir, xdebugtrace.Limits{
				MaxUpload: cfg.XdebugTrace.MaxUploadBytes,
				MaxSize:   cfg.XdebugTrace.MaxSizeBytes,
				Keep:      cfg.XdebugTrace.Keep,
			}))
		}
	}
	if enabled.IsEnabled("monolog") {
		registry.Register(monologMod)
	}
	if enabled.IsEnabled("smtp") {
		registry.Register(smtpMod)
	}
	if enabled.IsEnabled("var-dump") {
		registry.Register(vardumperMod)
	}
	if enabled.IsEnabled("sms") {
		registry.Register(sms.New())
	}
	// Xdebug proxy — off by default; forwards DBGp to the IDE, adds read-only
	// MCP tools and the /api/xdebug routes + "xdebug" WebSocket channel for the UI.
	if cfg.Xdebug.Enabled {
		xdebugMod := xdebug.New(xdebug.Config{
			Addr:      cfg.Xdebug.Addr,
			IDEAddr:   cfg.Xdebug.IDEAddr,
			AllowEval: cfg.Xdebug.AllowEval,

			AllowControl: cfg.Xdebug.AllowControlOn(),

			Mode:             cfg.Xdebug.Mode,
			PauseOnFirstLine: cfg.Xdebug.PauseOnFirstLine,
			IdleTimeout:      cfg.Xdebug.IdleTimeoutDur,
		})
		xdebugMod.SetBroadcaster(hub)
		hub.AddServerChannel(xdebug.Channel)
		registry.Register(xdebugMod)
	}
	// HTTP proxy — stores events as http-dump with response data.
	registry.Register(proxyMod)

	// Register webhooks module if any webhooks are configured.
	if len(cfg.Webhooks) > 0 {
		whConfigs := make([]webhooks.WebhookConfig, len(cfg.Webhooks))
		for i, w := range cfg.Webhooks {
			retry := true
			if w.Retry != nil {
				retry = *w.Retry
			}
			verifySSL := false
			if w.VerifySSL != nil {
				verifySSL = *w.VerifySSL
			}
			whConfigs[i] = webhooks.WebhookConfig{
				Event:     w.Event,
				URL:       w.URL,
				Headers:   w.Headers,
				VerifySSL: verifySSL,
				Retry:     retry,
			}
		}
		registry.Register(webhooks.New(whConfigs, collector))
		slog.Info("webhooks registered", "count", len(whConfigs))
	}

	var httpDumpMod *httpdumps.Module
	if enabled.IsEnabled("http-dump") {
		httpDumpMod = httpdumps.New(attachments, db)
		httpDumpMod.SetMockRules(append(mockRules(cfg.Mocks), openAPIMockRules(cfg.MocksOpenAPI)...))
		httpDumpMod.SetMCPMockWrite(cfg.MCP.MockWriteOn())
		registry.Register(httpDumpMod) // catch-all, must be last
	}

	// Modules that send notices of their own (module.Broadcasting) get the hub.
	registry.WireBroadcaster(hub)

	// Events removed by delete or clear-all take their files and module data
	// with them.
	store.OnDelete(func(uuids []string) {
		for _, u := range uuids {
			attachments.DeleteByEvent(u)
		}
		registry.NotifyEventsDeleted(uuids)
	})

	// Build event service and inject into TCP modules.
	eventService := httpserver.NewEventService(store, hub, registry, collector, db)
	if enabled.IsEnabled("monolog") {
		monologMod.SetEventService(eventService)
	}
	if enabled.IsEnabled("smtp") {
		smtpMod.SetEventService(eventService)
	}
	if enabled.IsEnabled("var-dump") {
		vardumperMod.SetEventService(eventService)
	}
	proxyMod.SetEventService(eventService)
	if httpDumpMod != nil {
		httpDumpMod.SetEventService(eventService)
	}

	application := app.New(cfg, db, registry, hub, store, attachments, collector)
	application.Run()
}

// openAPIMockRules reads the specs of the mocks_openapi section. A spec that
// cannot be read is logged and skipped, so a bad file never stops the server.
func openAPIMockRules(defs []app.MockOpenAPIDef) []httpdumps.MockRule {
	var rules []httpdumps.MockRule
	for _, d := range defs {
		res, err := httpdumps.OpenAPIRulesFromFile(d.File, httpdumps.OpenAPIImport{Project: d.Project, Group: d.Group, BasePath: d.BasePath})
		if err != nil {
			slog.Warn("mock: skipping OpenAPI spec", "file", d.File, "err", err)
			continue
		}
		for _, sk := range res.Skipped {
			slog.Warn("mock: OpenAPI operation skipped", "file", d.File, "method", sk.Method, "path", sk.Path, "reason", sk.Reason)
		}
		slog.Info("mock: OpenAPI spec loaded", "file", d.File, "group", res.Group, "rules", len(res.Rules))
		rules = append(rules, res.Rules...)
	}
	return rules
}

// mockRules converts the mocks section of buggregator.yaml. A delay that is
// not a duration is logged and dropped; the rest is validated by the module.
func mockRules(defs []app.MockDef) []httpdumps.MockRule {
	rules := make([]httpdumps.MockRule, 0, len(defs))
	for _, d := range defs {
		r := httpdumps.MockRule{
			Name: d.Name, Method: d.Method, Path: d.Path, Status: d.Status,
			Headers: d.Headers, Body: d.Body, Project: d.Project,
			Template: d.Template, Seed: d.Seed,
		}
		if d.Delay != "" {
			if delay, err := time.ParseDuration(d.Delay); err == nil {
				r.DelayMs = int(delay / time.Millisecond)
			} else {
				slog.Warn("mock: delay is not a duration, ignored", "path", d.Path, "delay", d.Delay)
			}
		}
		rules = append(rules, r)
	}
	return rules
}
