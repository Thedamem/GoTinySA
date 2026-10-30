// SPDX-License-Identifier: GPL-3.0-or-later

// Package webui serves a small mobile-friendly page for the running tinySA
// session: live spectrum, waterfall and the controls needed to tune an antenna
// from a phone on the same local network.
package webui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"tinysa_go/internal/tinysa"
)

// Backend is the part of the desktop application the web UI needs. It is an
// interface so the HTTP layer can be tested without a real tinySA.
type Backend interface {
	Status() Status
	Ports() ([]Port, error)
	Connect(port string) error
	CancelConnect()
	Disconnect() error
	ApplySweep(cfg tinysa.SweepConfig) error
	StopSweep() error
	Frame() *tinysa.Frame
}

// Status is the connection state shown in the browser. Config is the last
// sweep configuration applied by either UI, so a phone that opens the page
// while the desktop is already sweeping starts with the same controls.
type Status struct {
	Connected  bool                `json:"connected"`
	Connecting bool                `json:"connecting"`
	Port       string              `json:"port"`
	Sweeping   bool                `json:"sweeping"`
	Model      tinysa.Model        `json:"model"`
	Battery    string              `json:"battery"`
	Version    string              `json:"version"`
	Config     *tinysa.SweepConfig `json:"config,omitempty"`
}

// Port describes one serial port for the browser's port picker.
type Port struct {
	Name     string `json:"name"`
	VID      string `json:"vid"`
	PID      string `json:"pid"`
	Serial   string `json:"serial"`
	Product  string `json:"product"`
	IsTinySA bool   `json:"isTinySA"`
}

// Server is a running HTTP server. Create one with Start and stop it with
// Shutdown.
type Server struct {
	backend Backend
	ln      net.Listener
	httpSrv *http.Server
	started time.Time
}

//go:embed assets
var assets embed.FS

// Start listens on addr (for example "0.0.0.0:8420") and serves the web
// interface in the background. The caller owns Shutdown.
func Start(backend Backend, addr string) (*Server, error) {
	if backend == nil {
		return nil, errors.New("webui: nil backend")
	}
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		backend: backend,
		ln:      ln,
		started: time.Now(),
	}
	s.httpSrv = &http.Server{
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		_ = s.httpSrv.Serve(ln)
	}()
	return s, nil
}

// Addr returns the listener address, useful in logs and tests.
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Port returns the actual TCP port (8420 unless a free port was requested).
func (s *Server) Port() int { return s.ln.Addr().(*net.TCPAddr).Port }

// URLs returns the current browser addresses, LAN interfaces first and
// loopback last. It is recomputed on every call so a Wi-Fi/hotspot switch is
// reflected in the desktop dialog.
func (s *Server) URLs() []string { return LANURLs(s.Port()) }

// Started reports when the server was started.
func (s *Server) Started() time.Time { return s.started }

// Shutdown gracefully stops the HTTP server. Active SSE clients may keep
// Shutdown waiting until ctx expires; Close is used by the app on exit.
func (s *Server) Shutdown(ctx context.Context) error { return s.httpSrv.Shutdown(ctx) }

// Close stops the listener and closes every active connection immediately.
func (s *Server) Close() error { return s.httpSrv.Close() }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/ports", s.handlePorts)
	mux.HandleFunc("/api/connect", s.handleConnect)
	mux.HandleFunc("/api/connect/cancel", s.handleCancelConnect)
	mux.HandleFunc("/api/disconnect", s.handleDisconnect)
	mux.HandleFunc("/api/sweep", s.handleApplySweep)
	mux.HandleFunc("/api/sweep/stop", s.handleStopSweep)
	mux.HandleFunc("/api/frame", s.handleFrame)
	mux.HandleFunc("/api/csv", s.handleCSV)
	mux.HandleFunc("/api/events", s.handleEvents)

	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		// The directory is embedded at compile time, so this cannot happen.
		panic(err)
	}
	static := http.FileServer(http.FS(sub))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A phone may keep the page open across app updates; revalidate the
		// small embedded assets instead of serving a stale JS/CSS pair.
		w.Header().Set("Cache-Control", "no-cache")
		static.ServeHTTP(w, r)
	}))

	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; "+
				"connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.backend.Status())
}

func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	ports, err := s.backend.Ports()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ports == nil {
		ports = []Port{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ports": ports})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Port string `json:"port"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Port = strings.TrimSpace(req.Port)
	if req.Port == "" {
		writeError(w, http.StatusBadRequest, "port is empty")
		return
	}
	if err := s.backend.Connect(req.Port); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleCancelConnect(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	s.backend.CancelConnect()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := s.backend.Disconnect(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleApplySweep(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var cfg tinysa.SweepConfig
	if err := decodeJSON(w, r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate before stopping a running sweep, so a bad request cannot
	// interrupt a measurement.
	st := s.backend.Status()
	if !st.Connected {
		writeError(w, http.StatusConflict, "not connected")
		return
	}
	if err := cfg.Validate(st.Model); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.backend.ApplySweep(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStopSweep(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := s.backend.StopSweep(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleFrame(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, s.backend.Frame())
}

func (s *Server) handleCSV(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	f := s.backend.Frame()
	if f == nil {
		writeError(w, http.StatusConflict, "no sweep data yet")
		return
	}
	filename := fmt.Sprintf("tinysa_sweep_%s.csv", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(f.CSV())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(event string, v any) bool {
		data, err := json.Marshal(v)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	lastStatus := s.backend.Status()
	if !send("status", lastStatus) {
		return
	}
	var lastSeq uint64
	if f := s.backend.Frame(); f != nil {
		lastSeq = f.Seq
		if !send("frame", f) {
			return
		}
	}

	tick := time.NewTicker(100 * time.Millisecond)
	keepalive := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-tick.C:
			f := s.backend.Frame()
			if f != nil && f.Seq != lastSeq {
				lastSeq = f.Seq
				if !send("frame", f) {
					return
				}
			}
			// Config and connection state change rarely, but when either UI
			// applies a sweep the other one should see it promptly. Comparing
			// whole snapshots keeps the stream quiet while idle.
			st := s.backend.Status()
			if !reflect.DeepEqual(st, lastStatus) {
				lastStatus = st
				if !send("status", st) {
					return
				}
			}
		}
	}
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// LANURLs returns http://<ip>:<port> for every usable IPv4 interface, private
// home-network addresses first, followed by loopback for local checks.
func LANURLs(port int) []string {
	type candidate struct {
		url   string
		score int
	}
	var items []candidate
	seen := make(map[string]bool)

	add := func(ip net.IP) {
		ip4 := ip.To4()
		if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() || ip4.IsUnspecified() {
			return
		}
		host := ip4.String()
		if seen[host] {
			return
		}
		seen[host] = true

		score := 3
		switch {
		case strings.HasPrefix(host, "192.168."):
			score = 0
		case strings.HasPrefix(host, "10."):
			score = 1
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			score = 2
		}
		items = append(items, candidate{
			url:   fmt.Sprintf("http://%s:%d", host, port),
			score: score,
		})
	}

	if ifaces, err := net.Interfaces(); err == nil {
		for _, ifi := range ifaces {
			if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := ifi.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					add(v.IP)
				case *net.IPAddr:
					add(v.IP)
				}
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score < items[j].score
		}
		return items[i].url < items[j].url
	})

	urls := make([]string, 0, len(items)+1)
	for _, item := range items {
		urls = append(urls, item.url)
	}
	urls = append(urls, fmt.Sprintf("http://127.0.0.1:%d", port))
	return urls
}
