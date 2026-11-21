// Package csp captures browser security reports: Content Security Policy
// violations (report-uri, application/csp-report) and Reporting API reports
// (report-to, application/reports+json: csp-violation, deprecation,
// intervention, crash, ...). Point a policy at
// http://<buggregator>/csp or /csp/<project>.
package csp

import (
	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
)

// Module is the CSP / Reporting API catcher.
type Module struct {
	module.BaseModule
}

// New returns the module.
func New() *Module { return &Module{} }

func (m *Module) Name() string { return "CSP reports" }
func (m *Module) Type() string { return "csp" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler { return &handler{} }

func (m *Module) PreviewMapper() event.PreviewMapper { return &previewMapper{} }
