package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// proxyServer implements tcp.Starter.
type proxyServer struct {
	addr         string
	eventService EventStorer
	ca           *tls.Certificate
	certCache    sync.Map // host -> *tls.Certificate
	listener     net.Listener
}

func newProxyServer(addr string, es EventStorer) *proxyServer {
	return &proxyServer{addr: addr, eventService: es}
}

func (s *proxyServer) Start(ctx context.Context) error {
	ca, err := generateCA()
	if err != nil {
		return fmt.Errorf("proxy: generate CA: %w", err)
	}
	s.ca = ca

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("proxy: listen %s: %w", s.addr, err)
	}
	s.listener = ln

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					slog.Error("proxy: accept error", "err", err)
					continue
				}
			}
			go s.handleConn(ctx, conn)
		}
	}()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	return nil
}

func (s *proxyServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *proxyServer) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	if req.Method == http.MethodConnect {
		s.handleConnect(ctx, conn, req)
	} else {
		s.handleHTTP(ctx, conn, req)
	}
}

// handleConnect handles HTTPS CONNECT tunneling with MITM.
func (s *proxyServer) handleConnect(ctx context.Context, conn net.Conn, connectReq *http.Request) {
	host := connectReq.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}
	hostname, _, _ := net.SplitHostPort(host)

	// Respond 200 to establish the tunnel.
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Get or create a TLS certificate for this host.
	cert, err := s.certForHost(hostname)
	if err != nil {
		slog.Error("proxy: cert generation failed", "host", hostname, "err", err)
		return
	}

	// TLS handshake with the client.
	tlsConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	if err := tlsConn.Handshake(); err != nil {
		slog.Debug("proxy: client TLS handshake failed", "host", hostname, "err", err)
		return
	}
	defer tlsConn.Close()

	clientReader := bufio.NewReader(tlsConn)

	// Handle multiple requests over the same connection (HTTP/1.1 keep-alive).
	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			return
		}

		req.URL.Scheme = "https"
		req.URL.Host = host
		req.RequestURI = ""

		s.proxyAndRecord(ctx, tlsConn, req, "https")
	}
}

// handleHTTP handles plain HTTP proxy requests.
func (s *proxyServer) handleHTTP(ctx context.Context, conn net.Conn, req *http.Request) {
	req.RequestURI = ""
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	s.proxyAndRecord(ctx, conn, req, "http")
}

// proxyAndRecord forwards the request, records the exchange, and writes the response back.
func (s *proxyServer) proxyAndRecord(ctx context.Context, w io.Writer, req *http.Request, scheme string) {
	start := time.Now()

	// Capture request body.
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(io.LimitReader(req.Body, maxBodySize+1))
		req.Body = io.NopCloser(strings.NewReader(string(reqBody)))
	}

	// Remove hop-by-hop headers.
	removeHopHeaders(req.Header)

	captured := &capturedRequest{
		Method:  req.Method,
		URI:     req.URL.Path,
		Host:    req.URL.Host,
		Scheme:  scheme,
		Headers: req.Header,
		Query:   req.URL.Query(),
		Body:    reqBody,
	}

	// Forward the request to the real server.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	resp, err := transport.RoundTrip(req)
	durationMs := float64(time.Since(start).Milliseconds())

	if err != nil {
		// Send 502 back to the client.
		fmt.Fprintf(w, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
		s.storeEvent(ctx, captured, nil, durationMs, err.Error())
		return
	}
	defer resp.Body.Close()

	// Capture response body.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))

	capturedResp := &capturedResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// Write the response back to the client.
	writeResponse(w, resp, respBody)

	s.storeEvent(ctx, captured, capturedResp, durationMs, "")
}

func writeResponse(w io.Writer, resp *http.Response, body []byte) {
	fmt.Fprintf(w, "HTTP/%d.%d %s\r\n", resp.ProtoMajor, resp.ProtoMinor, resp.Status)
	resp.Header.Del("Transfer-Encoding")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	resp.Header.Write(w)
	fmt.Fprintf(w, "\r\n")
	w.Write(body)
}

func (s *proxyServer) storeEvent(ctx context.Context, req *capturedRequest, resp *capturedResponse, durationMs float64, proxyErr string) {
	inc := buildHTTPDumpEvent(req, resp, durationMs, proxyErr)
	if err := s.eventService.HandleIncoming(ctx, inc); err != nil {
		slog.Error("proxy: failed to store event", "err", err)
	}
}

func (s *proxyServer) certForHost(host string) (*tls.Certificate, error) {
	if cached, ok := s.certCache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}
	cert, err := generateHostCert(s.ca, host)
	if err != nil {
		return nil, err
	}
	s.certCache.Store(host, cert)
	return cert, nil
}

var hopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate",
	"Proxy-Authorization", "Te", "Trailer",
	"Transfer-Encoding", "Upgrade",
}

func removeHopHeaders(h http.Header) {
	for _, hdr := range hopHeaders {
		h.Del(hdr)
	}
}

// parseRequestURI extracts the path from an absolute proxy URI.
func parseRequestURI(rawURI string) string {
	u, err := url.Parse(rawURI)
	if err != nil {
		return rawURI
	}
	return u.Path
}
