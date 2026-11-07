package xdebug

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP control tools (xdebug.allow_control): an agent steps, continues, stops
// or releases a session Buggregator drives (no IDE), and waits for a pause.
// While an IDE is attached they are refused: the IDE drives.

const (
	defaultStepWait  = 30 * time.Second
	maxStepWait      = 30 * time.Second
	defaultPauseWait = 60 * time.Second
	maxPauseWait     = 60 * time.Second
	pollEvery        = 25 * time.Millisecond
)

type stepInput struct {
	SessionID      string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	Mode           string `json:"mode,omitempty" jsonschema:"into, over (default) or out"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"How long to wait for the next pause, 1-30 (default 30)"`
}

type continueInput struct {
	SessionID      string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"How long to wait for the next pause, 1-30 (default 30)"`
}

type stopInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	Mode      string `json:"mode,omitempty" jsonschema:"detach (default): let the script run to its end without the debugger; stop: end the PHP request now (partial work may be left behind)"`
}

type waitInput struct {
	SessionID      string `json:"session_id,omitempty" jsonschema:"Wait for this session only; empty = any session"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"How long to wait, 1-60 (default 60)"`
}

// waitSeconds clamps a tool's timeout_seconds.
func waitSeconds(n int, def, max time.Duration) time.Duration {
	if n <= 0 {
		return def
	}
	if n >= int(max/time.Second) { // clamp first: a huge n overflows Duration
		return max
	}
	return time.Duration(n) * time.Second
}

// agentName is the MCP client's name, shown in the UI as who drives.
func agentName(req *sdkmcp.CallToolRequest) string {
	if req == nil || req.Session == nil {
		return ""
	}
	if ip := req.Session.InitializeParams(); ip != nil && ip.ClientInfo != nil {
		return ip.ClientInfo.Name
	}
	return ""
}

// pauseSeqOf is the session's pause counter: it grows on every new break.
func (s *Session) pauseSeqOf() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pauseSeq
}

// waitAfter waits until the session pauses again (pause counter past seq) or
// ends. ok is false on timeout.
func (s *Session) waitAfter(ctx context.Context, seq int, d time.Duration) (bool, error) {
	t := time.NewTimer(d)
	defer t.Stop()
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		s.mu.Lock()
		done := s.closed || (s.status == "break" && s.conts == 0 && s.pauseSeq > seq)
		s.mu.Unlock()
		if done {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-t.C:
			return false, nil
		case <-tick.C:
		}
	}
}

// stopped describes where a session is after a continuation: the new
// location and source when paused, the state when it ended or still runs.
func (s *Session) stopped(waited bool) (map[string]any, string) {
	in := s.Info()
	out := map[string]any{"session_id": s.ID, "state": in.State}
	switch {
	case in.State == "break":
		out["file"], out["line"] = in.LastFile, in.LastLine
		if frames, err := s.stack(); err == nil && len(frames) > 0 {
			out["where"] = frames[0].Where
		}
		if src, err := s.sourceLines(in.LastFile, in.LastLine-5, in.LastLine+5); err == nil {
			out["source"] = src
		}
		out["idle_deadline"] = s.Info().IdleDeadline
		return out, fmt.Sprintf("%s:%d", baseName(in.LastFile), in.LastLine)
	case in.ClosedReason != "":
		out["closed_reason"] = in.ClosedReason
		return out, "ended: " + in.ClosedReason
	case !waited:
		out["note"] = "still running: no pause within the timeout; call xdebug_wait or xdebug_sessions later"
	}
	return out, in.State
}

func registerControlTools(server *sdkmcp.Server, p *Proxy, run runFunc) {
	cont := func(ctx context.Context, req *sdkmcp.CallToolRequest, sessionID, tool, cmd string, wait time.Duration) (*sdkmcp.CallToolResult, any, error) {
		return run(sessionID, tool, cmd, func(s *Session) (any, string, error) {
			seq := s.pauseSeqOf()
			if err := s.Continue(cmd, ByAgent, agentName(req)); err != nil {
				return nil, "", err
			}
			ok, err := s.waitAfter(ctx, seq, wait)
			if err != nil {
				return nil, "", err
			}
			out, summary := s.stopped(ok)
			return out, summary, nil
		})
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_step",
		Description: "Step a paused Xdebug session that Buggregator drives (no IDE attached): mode into, over (default) or out. Waits for the next pause (up to 30 s) and returns the new file, line, function and source around it, or that the script ended. Refused while an IDE drives the session.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in stepInput) (*sdkmcp.CallToolResult, any, error) {
		cmd := map[string]string{"": "step_over", "over": "step_over", "into": "step_into", "out": "step_out"}[in.Mode]
		if cmd == "" {
			return nil, nil, fmt.Errorf("mode must be into, over or out")
		}
		return cont(ctx, req, in.SessionID, "xdebug_step", cmd, waitSeconds(in.TimeoutSeconds, defaultStepWait, maxStepWait))
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_continue",
		Description: "Resume a paused Xdebug session that Buggregator drives (no IDE attached) until the next breakpoint or xdebug_break(). Waits up to 30 s and returns the new location, or that the script ended or still runs. Refused while an IDE drives the session.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in continueInput) (*sdkmcp.CallToolResult, any, error) {
		return cont(ctx, req, in.SessionID, "xdebug_continue", "run", waitSeconds(in.TimeoutSeconds, defaultStepWait, maxStepWait))
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_stop",
		Description: "End debugging of a paused Xdebug session that Buggregator drives. mode detach (default, safe): the script runs to its end without the debugger. mode stop: the PHP request ends at once and may leave partial work behind. Refused while an IDE drives the session.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in stopInput) (*sdkmcp.CallToolResult, any, error) {
		cmd := map[string]string{"": "detach", "detach": "detach", "stop": "stop"}[in.Mode]
		if cmd == "" {
			return nil, nil, fmt.Errorf("mode must be detach or stop")
		}
		return cont(ctx, req, in.SessionID, "xdebug_stop", cmd, 5*time.Second)
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_wait",
		Description: "Wait until an Xdebug session is paused (up to 60 s) and return it with its location. Use it after setting a breakpoint and firing the request (e.g. with curl). Returns at once when a session is already paused.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in waitInput) (*sdkmcp.CallToolResult, any, error) {
		s, err := p.waitPaused(ctx, in.SessionID, waitSeconds(in.TimeoutSeconds, defaultPauseWait, maxPauseWait))
		if err != nil {
			return nil, nil, err
		}
		return run(s.ID, "xdebug_wait", "", func(s *Session) (any, string, error) {
			out, summary := s.stopped(true)
			out["driver"] = s.Info().Driver
			return out, summary, nil
		})
	})
}

// waitPaused returns a paused session (id, or any when empty), waiting up
// to d for one. Several paused at once: the most recently paused.
func (p *Proxy) waitPaused(ctx context.Context, id string, d time.Duration) (*Session, error) {
	t := time.NewTimer(d)
	defer t.Stop()
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	for {
		var best *Session
		var bestAt time.Time
		for _, s := range p.Sessions() {
			if id != "" && s.ID != id {
				continue
			}
			if s.Paused() {
				if at := s.pausedAtOf(); best == nil || at.After(bestAt) {
					best, bestAt = s, at
				}
			}
		}
		if best != nil {
			return best, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
			return nil, errors.New("no session paused within the timeout: is Xdebug pointed at Buggregator and the request started?")
		case <-tick.C:
		}
	}
}

func (s *Session) pausedAtOf() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pausedAt
}

type bpSetInput struct {
	File      string `json:"file,omitempty" jsonschema:"File as PHP sees it: an absolute path (/var/www/app/src/User.php) or a file:// URI. Paths on the PHP side, not your local checkout, unless they are the same."`
	Line      int    `json:"line,omitempty" jsonschema:"Line number (1-based) for a line breakpoint"`
	Exception string `json:"exception,omitempty" jsonschema:"Exception class for an exception breakpoint instead of file+line, e.g. Throwable or App\\Exceptions\\PaymentFailed (a leading \\ is dropped)"`
	Condition string `json:"condition,omitempty" jsonschema:"PHP expression; stop only when it is true. Needs xdebug.allow_eval."`
}

type bpRemoveInput struct {
	ID int64 `json:"id" jsonschema:"Breakpoint id from xdebug_breakpoints"`
}

// registerBreakpointTools: listing is a read (always on); set and remove
// change what the app does and come with allow_control.
func registerBreakpointTools(server *sdkmcp.Server, p *Proxy, control bool) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_breakpoints",
		Description: "Breakpoints Buggregator saves and sends to every Xdebug session it drives (no IDE attached; an IDE sets its own). With session_id of a session Buggregator drives: each with its state in that session (resolved, unresolved, skipped = has a condition while eval is off, disabled, pending = sent at the next pause), resolved_line and hit_count.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sessionInput) (*sdkmcp.CallToolResult, any, error) {
		list := p.bps.List()
		if in.SessionID == "" {
			return textResult(list), nil, nil
		}
		s, err := p.Session(in.SessionID)
		if err != nil {
			return nil, nil, err
		}
		if s.Info().Driver != DriverHere {
			// An IDE drives it: its breakpoints are the IDE's, not the saved ones.
			bps, err := s.ideBreakpoints()
			if err != nil {
				return nil, nil, err
			}
			return textResult(map[string]any{"driver": DriverIDE, "ide_breakpoints": bps}), nil, nil
		}
		return textResult(s.SessionBreakpoints(list)), nil, nil
	})
	if !control {
		return
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_breakpoint_set",
		Description: "Save a breakpoint: file + line, or an exception class. Paused sessions Buggregator drives have it when this returns; running ones get it at their next pause; every new session gets it before its script runs. Then fire the request and call xdebug_wait.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in bpSetInput) (*sdkmcp.CallToolResult, any, error) {
		bp, err := p.AddBreakpoint(SavedBreakpoint{File: in.File, Line: in.Line, Exception: in.Exception, Condition: strings.TrimSpace(in.Condition), Enabled: true, Source: "mcp"})
		if err != nil {
			return nil, nil, err
		}
		return textResult(bp), nil, nil
	})
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_breakpoint_remove",
		Description: "Delete a saved breakpoint by id (see xdebug_breakpoints), also from live sessions Buggregator drives.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in bpRemoveInput) (*sdkmcp.CallToolResult, any, error) {
		if err := p.RemoveBreakpoint(in.ID); err != nil {
			return nil, nil, err
		}
		return textResult(map[string]any{"removed": in.ID}), nil, nil
	})
}
