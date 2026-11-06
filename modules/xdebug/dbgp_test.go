package xdebug

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestReadPacket(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "length past body", in: "5\x00<a/>\x00", wantErr: true},
		{name: "missing trailing NUL", in: "4\x00<a/>x", wantErr: true},
		{name: "exact", in: "4\x00<a/>\x00", want: "<a/>"},
		{name: "bad length byte", in: "4x\x00<a/>\x00", wantErr: true},
		{name: "empty length", in: "\x00<a/>\x00", wantErr: true},
		{name: "too many digits", in: "99999999999\x00", wantErr: true},
		{name: "over cap", in: "99999999\x00", wantErr: true},
		{name: "short body", in: "10\x00<a/>", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readPacket(bufio.NewReader(strings.NewReader(tt.in)))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil || string(got) != tt.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestReadPacket_OverCapIsTooLarge(t *testing.T) {
	_, err := readPacket(bufio.NewReader(strings.NewReader("99999999\x00")))
	if !errors.Is(err, errPacketTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestEncodePacketRoundTrip(t *testing.T) {
	body := []byte(`<response transaction_id="1">é</response>`)
	got, err := readPacket(bufio.NewReader(bytes.NewReader(encodePacket(body))))
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReadCommand_LongLine(t *testing.T) {
	long := "eval -i 1 -- " + strings.Repeat("A", 10000)
	r := bufio.NewReaderSize(strings.NewReader(long+"\x00status -i 2\x00"), 16)
	if got, err := readCommand(r); err != nil || got != long {
		t.Fatalf("first: %v", err)
	}
	if got, err := readCommand(r); err != nil || got != "status -i 2" {
		t.Fatalf("second: %q %v", got, err)
	}
	huge := strings.Repeat("A", maxCommandSize+1) + "\x00"
	if _, err := readCommand(bufio.NewReader(strings.NewReader(huge))); !errors.Is(err, errPacketTooLarge) {
		t.Fatalf("huge: %v", err)
	}
}

func TestCommandTID(t *testing.T) {
	tests := []struct {
		line, name, tid string
		ok              bool
	}{
		{"run -i 5", "run", "5", true},
		{"property_get -n $x -i 12 -d 0", "property_get", "12", true},
		{`breakpoint_set -t line -f "file:///a -i 9.php" -i 3 -n 1`, "breakpoint_set", "3", true},
		{"eval -- LWkgNA== -i 4", "eval", "", false},
		{"status", "status", "", false},
	}
	for _, tt := range tests {
		name, s, e, ok := commandTID(tt.line)
		if name != tt.name || ok != tt.ok || (ok && tt.line[s:e] != tt.tid) {
			t.Errorf("%q: name=%q ok=%v tid=%q", tt.line, name, ok, tt.line[s:e])
		}
	}
}

func TestParseRootAndRewrite(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="iso-8859-1"?>
<response xmlns="urn:debugger_protocol_v1" command="eval" transaction_id="17" status="break"><property name="x" transaction_id="17"/></response>`)
	rt, err := parseRoot(body)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Name != "response" || rt.TransactionID != "17" || rt.Status != "break" || rt.Command != "eval" {
		t.Fatalf("root: %+v", rt)
	}
	out := string(rt.withTID(body, "3"))
	if !strings.Contains(out, `command="eval" transaction_id="3"`) || !strings.Contains(out, `<property name="x" transaction_id="17"/>`) {
		t.Fatalf("rewrite touched more than the root: %s", out)
	}
}

func TestQuoteArg(t *testing.T) {
	for in, want := range map[string]string{
		"$x":          "$x",
		`$a["k y"]`:   `"$a[\"k y\"]"`,
		`file:///a b`: `"file:///a b"`,
		"":            `""`,
		`back\slash`:  `"back\\slash"`,
	} {
		if got := quoteArg(in); got != want {
			t.Errorf("quoteArg(%q) = %q, want %q", in, got, want)
		}
	}
}
