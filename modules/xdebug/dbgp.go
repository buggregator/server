package xdebug

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// DBGp framing (https://xdebug.org/docs/dbgp):
//   engine → client: "<decimal length>\0<xml>\0"
//   client → engine: "command -i <tid> [args] [-- base64]\0"

const (
	maxPacketSize   = 32 << 20 // engine packets (source, big arrays) above this are a protocol error
	maxCommandSize  = 1 << 20  // client command lines
	maxLengthDigits = 10
)

var errPacketTooLarge = errors.New("dbgp: packet too large")

// readPacket reads one engine packet and returns its XML body.
func readPacket(r *bufio.Reader) ([]byte, error) {
	var n int
	digits := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0 {
			break
		}
		if b < '0' || b > '9' {
			return nil, fmt.Errorf("dbgp: bad length byte %q", b)
		}
		digits++
		if digits > maxLengthDigits {
			return nil, errPacketTooLarge
		}
		n = n*10 + int(b-'0')
	}
	if digits == 0 {
		return nil, errors.New("dbgp: empty length")
	}
	if n > maxPacketSize {
		return nil, errPacketTooLarge
	}
	buf := make([]byte, n+1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if buf[n] != 0 {
		return nil, errors.New("dbgp: packet not NUL-terminated")
	}
	return buf[:n], nil
}

// encodePacket frames an engine packet.
func encodePacket(body []byte) []byte {
	out := make([]byte, 0, len(body)+12)
	out = strconv.AppendInt(out, int64(len(body)), 10)
	out = append(out, 0)
	out = append(out, body...)
	return append(out, 0)
}

// readCommand reads one NUL-terminated client command (without the NUL).
func readCommand(r *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		chunk, err := r.ReadSlice(0)
		if sb.Len()+len(chunk) > maxCommandSize {
			return "", errPacketTooLarge
		}
		if err == nil {
			sb.Write(chunk[:len(chunk)-1])
			return sb.String(), nil
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			sb.Write(chunk)
			continue
		}
		return "", err
	}
}

// commandTID returns the command name and the byte span of the -i value.
// The span is found outside quoted values and before the " -- " data part,
// so base64 data and quoted file names are never touched.
func commandTID(line string) (name string, start, end int, ok bool) {
	head := line
	if i := strings.Index(line, " -- "); i >= 0 {
		head = line[:i]
	}
	type tok struct{ s, e int }
	var toks []tok
	i := 0
	for i < len(head) {
		for i < len(head) && head[i] == ' ' {
			i++
		}
		if i >= len(head) {
			break
		}
		s := i
		if head[i] == '"' {
			i++
			for i < len(head) && head[i] != '"' {
				if head[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(head) {
				i++
			}
		} else {
			for i < len(head) && head[i] != ' ' {
				i++
			}
		}
		toks = append(toks, tok{s, i})
	}
	if len(toks) == 0 {
		return "", 0, 0, false
	}
	name = head[toks[0].s:toks[0].e]
	for k := 1; k+1 < len(toks); k++ {
		if head[toks[k].s:toks[k].e] == "-i" {
			return name, toks[k+1].s, toks[k+1].e, true
		}
	}
	return name, 0, 0, false
}

// rootTag is the first element of an engine packet.
type rootTag struct {
	Name          string // init, response, stream, notify
	TransactionID string
	Status        string
	Command       string
	tidStart      int // byte span of the transaction_id value, -1 when absent
	tidEnd        int
}

var tidAttr = regexp.MustCompile(`\stransaction_id=("([^"]*)"|'([^']*)')`)

// parseRoot reads the root element's name and attributes.
func parseRoot(body []byte) (rootTag, error) {
	rt := rootTag{tidStart: -1}
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = passthroughCharset
	for {
		t, err := dec.Token()
		if err != nil {
			return rt, fmt.Errorf("dbgp: no root element: %w", err)
		}
		se, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		rt.Name = se.Name.Local
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "transaction_id":
				rt.TransactionID = a.Value
			case "status":
				rt.Status = a.Value
			case "command":
				rt.Command = a.Value
			}
		}
		break
	}
	if rt.TransactionID == "" {
		return rt, nil
	}
	// Locate the attribute inside the root start tag for in-place rewriting.
	open := bytes.Index(body, []byte("<"+rt.Name))
	if open < 0 {
		return rt, nil
	}
	closeIdx := bytes.IndexByte(body[open:], '>')
	if closeIdx < 0 {
		return rt, nil
	}
	tag := body[open : open+closeIdx]
	m := tidAttr.FindSubmatchIndex(tag)
	if m == nil {
		return rt, nil
	}
	if m[4] >= 0 {
		rt.tidStart, rt.tidEnd = open+m[4], open+m[5]
	} else {
		rt.tidStart, rt.tidEnd = open+m[6], open+m[7]
	}
	return rt, nil
}

// withTID returns body with the root transaction_id value replaced.
func (rt rootTag) withTID(body []byte, tid string) []byte {
	if rt.tidStart < 0 {
		return body
	}
	out := make([]byte, 0, len(body)+len(tid))
	out = append(out, body[:rt.tidStart]...)
	out = append(out, tid...)
	return append(out, body[rt.tidEnd:]...)
}

// passthroughCharset lets encoding/xml read Xdebug's iso-8859-1 prolog.
// Names and attributes we read are ASCII; values come base64-encoded.
func passthroughCharset(_ string, input io.Reader) (io.Reader, error) {
	return input, nil
}

// quoteArg quotes a DBGp argument value when needed.
func quoteArg(s string) string {
	if s != "" && !strings.ContainsAny(s, " \"\\\x00") {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
