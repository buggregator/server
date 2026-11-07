package xdebug

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTP API for the Xdebug page. Every DBGp read goes through the session's
// command queue and allow-list; UI reads are not written to the agent log and
// are cached per pause. Control routes (run, step, stop, detach) work only on
// a session driven here, never while an IDE drives it.

type apiError struct {
	Code  string `json:"code"`
	Error string `json:"error"`
	State string `json:"state,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // source code and PHP names stay readable
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code string, err error) {
	writeJSON(w, status, apiError{Code: code, Error: err.Error()})
}

// registerAPI adds the /api/xdebug routes; wrap is the auth middleware.
func (m *Module) registerAPI(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	h := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, wrap(fn))
	}
	h("GET /api/xdebug", m.handleStatus)
	h("GET /api/xdebug/sessions", m.handleSessions)
	h("GET /api/xdebug/sessions/{id}", m.withSession(m.handleSession))
	h("DELETE /api/xdebug/sessions/{id}", m.handleDismiss)
	h("GET /api/xdebug/sessions/{id}/stack", m.withSession(m.handleStack))
	h("GET /api/xdebug/sessions/{id}/contexts", m.withSession(m.handleContexts))
	h("GET /api/xdebug/sessions/{id}/variables", m.withSession(m.handleVariables))
	h("GET /api/xdebug/sessions/{id}/property", m.withSession(m.handleProperty))
	h("GET /api/xdebug/sessions/{id}/source", m.withSession(m.handleSource))
	h("GET /api/xdebug/sessions/{id}/breakpoints", m.withSession(m.handleBreakpoints))
	h("POST /api/xdebug/sessions/{id}/eval", m.withSession(m.handleEval))
	h("GET /api/xdebug/sessions/{id}/agent-log", m.withSession(m.handleAgentLog))
	h("PUT /api/xdebug/settings", m.handleSettings)
	h("GET /api/xdebug/breakpoints", m.handleBPList)
	h("POST /api/xdebug/breakpoints", m.handleBPCreate)
	h("PATCH /api/xdebug/breakpoints/{bp}", m.handleBPUpdate)
	h("DELETE /api/xdebug/breakpoints/{bp}", m.handleBPDelete)
	h("GET /api/xdebug/files", m.handleFiles)
	for route, cmd := range controlRoutes {
		h("POST /api/xdebug/sessions/{id}/"+route, m.withSession(m.handleControl(cmd)))
	}
}

// controlRoutes maps the control routes to DBGp continuations.
var controlRoutes = map[string]string{
	"run": "run", "step-into": "step_into", "step-over": "step_over", "step-out": "step_out",
	"stop": "stop", "detach": "detach",
}

// jsonOnly guards the xdebug writes (eval, control, settings) when auth is
// off. A cross-site form or text/plain POST needs no CORS preflight, so the
// body must be application/json. And the default cors_origins ["*"] answers
// any preflight, so a request a browser marks Sec-Fetch-Site: cross-site
// (another site's page) is refused too; same-site (another port of
// localhost, the dev server) and non-browser clients pass.
func jsonOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		writeErr(w, http.StatusForbidden, "cross_site", errors.New("cross-site requests cannot change a debug session"))
		return false
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeErr(w, http.StatusUnsupportedMediaType, "unsupported_media_type", errors.New("Content-Type must be application/json"))
		return false
	}
	return true
}

func (m *Module) handleControl(cmd string) func(http.ResponseWriter, *http.Request, *Session) {
	return func(w http.ResponseWriter, r *http.Request, s *Session) {
		if !jsonOnly(w, r) {
			return
		}
		err := s.Continue(cmd, ByUI, "")
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, s.Info())
		case errors.Is(err, errIDEAttached):
			writeJSON(w, http.StatusConflict, apiError{Code: "ide_attached", Error: err.Error(), State: s.Info().State})
		case errors.Is(err, errNotPaused), errors.Is(err, errSessionClosed):
			writeJSON(w, http.StatusConflict, apiError{Code: "not_paused", Error: errNotPaused.Error(), State: s.Info().State})
		default:
			writeErr(w, http.StatusBadGateway, "xdebug_failed", err)
		}
	}
}

// settingsView is the settings part of GET /api/xdebug and PUT /api/xdebug/settings.
func (m *Module) settingsView() map[string]any {
	st := m.proxy.Settings()
	return map[string]any{
		"mode": st.Mode, "pause_on_first_line": st.PauseOnFirstLine,
		"idle_timeout_seconds": int(m.proxy.idleTimeout / time.Second),
	}
}

// handleSettings changes mode and pause_on_first_line for new sessions; a
// field left out keeps its value. Settings live in memory: a restart goes
// back to the config.
func (m *Module) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !jsonOnly(w, r) {
		return
	}
	var body struct {
		Mode             *string `json:"mode"`
		PauseOnFirstLine *bool   `json:"pause_on_first_line"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || (body.Mode == nil && body.PauseOnFirstLine == nil) {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New(`body must be {"mode": "proxy|auto|standalone", "pause_on_first_line": bool}`))
		return
	}
	st := m.proxy.Settings()
	if body.Mode != nil {
		st.Mode = *body.Mode
	}
	if body.PauseOnFirstLine != nil {
		st.PauseOnFirstLine = *body.PauseOnFirstLine
	}
	if err := m.proxy.SetSettings(st); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err)
		return
	}
	writeJSON(w, http.StatusOK, m.settingsView())
}

func (m *Module) handleStatus(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"enabled": true, "addr": m.cfg.Addr, "ide_addr": m.cfg.IDEAddr, "allow_eval": m.cfg.AllowEval,
		"allow_control": m.cfg.AllowControl,
	}
	for k, v := range m.settingsView() {
		out[k] = v
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) handleSessions(w http.ResponseWriter, r *http.Request) {
	infos := []Info{}
	for _, s := range m.proxy.Sessions() {
		infos = append(infos, s.Info())
	}
	writeJSON(w, http.StatusOK, infos)
}

func (m *Module) withSession(fn func(http.ResponseWriter, *http.Request, *Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := m.proxy.Session(r.PathValue("id"))
		if err != nil {
			writeErr(w, http.StatusNotFound, "not_found", err)
			return
		}
		fn(w, r, s)
	}
}

// readFailed maps a read error to the API's status codes.
func readFailed(w http.ResponseWriter, s *Session, err error) {
	switch {
	case errors.Is(err, errNotPaused), errors.Is(err, errSessionClosed):
		writeJSON(w, http.StatusConflict, apiError{Code: "not_paused", Error: errNotPaused.Error(), State: s.Info().State})
	case errors.Is(err, errCommandTimeout):
		writeErr(w, http.StatusGatewayTimeout, "timeout", err)
	default:
		var de *dbgpError
		if errors.As(err, &de) {
			writeErr(w, http.StatusUnprocessableEntity, "xdebug_error", err)
			return
		}
		writeErr(w, http.StatusBadGateway, "xdebug_failed", err)
	}
}

func (m *Module) handleSession(w http.ResponseWriter, r *http.Request, s *Session) {
	writeJSON(w, http.StatusOK, s.Info())
}

func (m *Module) handleDismiss(w http.ResponseWriter, r *http.Request) {
	err := m.proxy.Dismiss(r.PathValue("id"))
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, errNoSession):
		writeErr(w, http.StatusNotFound, "not_found", err)
	default:
		writeErr(w, http.StatusConflict, "session_live", err)
	}
}

// StackFrame is a frame for the UI; vendor frames can be folded.
type StackFrame struct {
	Frame
	Vendor bool `json:"vendor"`
}

func (m *Module) handleStack(w http.ResponseWriter, r *http.Request, s *Session) {
	resp, err := s.readResponse("stack_get", "")
	if err != nil {
		readFailed(w, s, err)
		return
	}
	frames := []StackFrame{}
	for _, f := range resp.Stack {
		m.proxy.files.see(f.Filename)
		frames = append(frames, StackFrame{
			Frame:  Frame{Level: f.Level, Where: f.Where, File: f.Filename, Line: f.Lineno, Type: f.Type},
			Vendor: strings.Contains(f.Filename, "/vendor/"),
		})
	}
	writeJSON(w, http.StatusOK, frames)
}

func (m *Module) handleContexts(w http.ResponseWriter, r *http.Request, s *Session) {
	frame, ok := intParam(w, r, "frame")
	if !ok {
		return
	}
	resp, err := s.readResponse("context_names", fmt.Sprintf("-d %d", frame))
	if err != nil {
		readFailed(w, s, err)
		return
	}
	type ctx struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	out := []ctx{}
	for _, c := range resp.Contexts {
		out = append(out, ctx{ID: c.ID, Name: c.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleVariables: the children depth is the IDE's max_depth setting (a
// session-wide feature the proxy never changes); expand with /property.
func (m *Module) handleVariables(w http.ResponseWriter, r *http.Request, s *Session) {
	frame, ok := intParam(w, r, "frame")
	if !ok {
		return
	}
	context, ok := intParam(w, r, "context")
	if !ok {
		return
	}
	resp, err := s.readResponse("context_get", fmt.Sprintf("-d %d -c %d", frame, context))
	if err != nil {
		readFailed(w, s, err)
		return
	}
	vars := []Var{}
	for _, p := range resp.Props {
		vars = append(vars, toVar(p))
	}
	writeJSON(w, http.StatusOK, vars)
}

func (m *Module) handleProperty(w http.ResponseWriter, r *http.Request, s *Session) {
	name := r.URL.Query().Get("name")
	if strings.TrimSpace(name) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New("name is required"))
		return
	}
	if strings.ContainsRune(name, 0) {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New("name must not contain NUL"))
		return
	}
	frame, ok := intParam(w, r, "frame")
	if !ok {
		return
	}
	context, ok := intParam(w, r, "context")
	if !ok {
		return
	}
	page, ok := intParam(w, r, "page")
	if !ok {
		return
	}
	args := fmt.Sprintf("-d %d -c %d -m %d -p %d -n %s", frame, context, maxPropertyData, page, quoteArg(name))
	resp, err := s.readResponse("property_get", args)
	if err != nil {
		readFailed(w, s, err)
		return
	}
	if len(resp.Props) == 0 {
		writeErr(w, http.StatusNotFound, "not_found", fmt.Errorf("property %s not found", name))
		return
	}
	writeJSON(w, http.StatusOK, toVar(resp.Props[0]))
}

// SourceLine is one line of the code view.
type SourceLine struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

const maxSourceLines = 2000

func (m *Module) handleSource(w http.ResponseWriter, r *http.Request, s *Session) {
	q := r.URL.Query()
	file := q.Get("file")
	if file == "" || strings.ContainsRune(file, 0) {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New("file is required"))
		return
	}
	begin, ok := intParam(w, r, "begin")
	if !ok {
		return
	}
	end, ok := intParam(w, r, "end")
	if !ok {
		return
	}
	if begin < 1 {
		begin = 1
	}
	if end < begin || end-begin > maxSourceLines {
		end = begin + maxSourceLines
	}
	resp, err := s.readResponse("source", fmt.Sprintf("-f %s -b %d -e %d", quoteArg(file), begin, end))
	if err != nil {
		readFailed(w, s, err)
		return
	}
	text := strings.TrimRight(decodeText(resp.Encoding, resp.Text), "\n")
	lines := []SourceLine{}
	if text != "" {
		for i, l := range strings.Split(text, "\n") {
			lines = append(lines, SourceLine{Line: begin + i, Text: strings.TrimRight(l, "\r")})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": file, "begin": begin, "lines": lines})
}

// Breakpoint is a breakpoint the IDE set; the UI shows gutter dots only.
type Breakpoint struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	State    string `json:"state"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
	HitCount int    `json:"hit_count"`
}

// handleBreakpoints: a session driven here lists the saved breakpoints with
// their state in it; a session an IDE drives lists what the IDE set.
func (m *Module) handleBreakpoints(w http.ResponseWriter, r *http.Request, s *Session) {
	if s.Info().Driver == DriverHere {
		writeJSON(w, http.StatusOK, s.SessionBreakpoints(m.proxy.bps.List()))
		return
	}
	resp, err := s.readResponse("breakpoint_list", "")
	if err != nil {
		readFailed(w, s, err)
		return
	}
	out := []Breakpoint{}
	for _, b := range resp.Breakpoints {
		out = append(out, Breakpoint{ID: b.ID, Type: b.Type, State: b.State, File: b.Filename, Line: b.Lineno, Function: b.Function, HitCount: b.HitCount})
	}
	writeJSON(w, http.StatusOK, out)
}

func (m *Module) handleEval(w http.ResponseWriter, r *http.Request, s *Session) {
	if !m.cfg.AllowEval {
		writeErr(w, http.StatusForbidden, "eval_disabled", errors.New("eval is off (xdebug.allow_eval)"))
		return
	}
	if !jsonOnly(w, r) {
		return
	}
	var body struct {
		Expression string `json:"expression"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil || strings.TrimSpace(body.Expression) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New(`body must be {"expression": "..."}`))
		return
	}
	v, err := s.eval(body.Expression) // never cached: an expression may change state
	if err != nil {
		readFailed(w, s, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (m *Module) handleAgentLog(w http.ResponseWriter, r *http.Request, s *Session) {
	writeJSON(w, http.StatusOK, s.AgentLog())
}

// intParam reads an optional non-negative integer query parameter (default 0).
func intParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return 0, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", fmt.Errorf("%s must be a non-negative integer", name))
		return 0, false
	}
	return n, true
}

// --- Saved breakpoints ---------------------------------------------------------

// bpBody is the body of POST and PATCH /api/xdebug/breakpoints; a field left
// out of a PATCH keeps its value.
type bpBody struct {
	Type      *string `json:"type"`
	File      *string `json:"file"`
	Line      *int    `json:"line"`
	Exception *string `json:"exception"`
	Condition *string `json:"condition"`
	Enabled   *bool   `json:"enabled"`
}

func (b bpBody) apply(bp *SavedBreakpoint) {
	if b.Type != nil {
		bp.Type = *b.Type
	}
	if b.File != nil {
		bp.File = *b.File
	}
	if b.Line != nil {
		bp.Line = *b.Line
	}
	if b.Exception != nil {
		bp.Exception = *b.Exception
	}
	if b.Condition != nil {
		bp.Condition = strings.TrimSpace(*b.Condition)
	}
	if b.Enabled != nil {
		bp.Enabled = *b.Enabled
	}
}

func readBPBody(w http.ResponseWriter, r *http.Request) (bpBody, bool) {
	var b bpBody
	if !jsonOnly(w, r) {
		return b, false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", errors.New(`body must be {"type", "file", "line", "exception", "condition", "enabled"}`))
		return b, false
	}
	return b, true
}

// bpFailed maps a breakpoint store error to a status.
func bpFailed(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errBPNotFound):
		writeErr(w, http.StatusNotFound, "not_found", err)
	case errors.Is(err, errBPCondition):
		writeErr(w, http.StatusForbidden, "eval_disabled", err)
	default:
		writeErr(w, http.StatusBadRequest, "bad_request", err)
	}
}

func (m *Module) handleBPList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, m.proxy.bps.List())
}

func (m *Module) handleBPCreate(w http.ResponseWriter, r *http.Request) {
	body, ok := readBPBody(w, r)
	if !ok {
		return
	}
	bp := SavedBreakpoint{Enabled: true, Source: "ui"}
	body.apply(&bp)
	bp, err := m.proxy.AddBreakpoint(bp)
	if err != nil {
		bpFailed(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bp)
}

func bpID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("bp"), 10, 64)
	if err != nil || id < 1 {
		writeErr(w, http.StatusNotFound, "not_found", errBPNotFound)
		return 0, false
	}
	return id, true
}

func (m *Module) handleBPUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := bpID(w, r)
	if !ok {
		return
	}
	body, ok := readBPBody(w, r)
	if !ok {
		return
	}
	bp, err := m.proxy.bps.Update(id, func(bp *SavedBreakpoint) error {
		body.apply(bp)
		if body.Condition != nil {
			return m.proxy.checkCondition(*bp)
		}
		return nil
	})
	if err != nil {
		bpFailed(w, err)
		return
	}
	m.proxy.BreakpointsChanged()
	writeJSON(w, http.StatusOK, bp)
}

func (m *Module) handleBPDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := bpID(w, r)
	if !ok {
		return
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		writeErr(w, http.StatusForbidden, "cross_site", errors.New("cross-site requests cannot change breakpoints"))
		return
	}
	if err := m.proxy.RemoveBreakpoint(id); err != nil {
		bpFailed(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleFiles offers files for a new breakpoint: seen in pauses and stacks
// (live sessions and tombstones) and of saved breakpoints.
func (m *Module) handleFiles(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 200 {
		limit = v
	}
	files := m.proxy.Files(r.URL.Query().Get("q"), limit)
	if files == nil {
		files = []string{}
	}
	writeJSON(w, http.StatusOK, files)
}
