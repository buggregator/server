// Package html captures events with the sender's own HTML markup: a report,
// a link to a tool's page, an experiment with a new kind of event. Send
//
//	curl -X POST http://html@127.0.0.1:8000 -H 'Content-Type: text/html' --data '<h1>Report</h1>'
//
// or JSON {"title": "…", "html": "…"} with X-Buggregator-Event: html, or POST
// to /html[/project]. The UI shows the markup in a sandboxed frame.
package html

import (
	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
)

// Module is the custom HTML event catcher.
type Module struct {
	module.BaseModule
}

// New returns the module.
func New() *Module { return &Module{} }

func (m *Module) Name() string { return "HTML" }
func (m *Module) Type() string { return "html" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler { return &handler{} }

func (m *Module) PreviewMapper() event.PreviewMapper { return &previewMapper{} }
