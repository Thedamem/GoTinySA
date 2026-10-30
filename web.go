// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"tinysa_go/internal/tinysa"
	ver "tinysa_go/internal/version"
	"tinysa_go/internal/webui"
)

// WebInfo describes the optional local web interface for the desktop UI.
type WebInfo struct {
	Running bool     `json:"running"`
	Port    int      `json:"port"`
	URLs    []string `json:"urls"`
	Started string   `json:"started"`
}

// StartWebServer starts the local web UI on port 8420. If the port is busy it
// falls back to a free port; the actual URL is returned for the dialog.
func (a *App) StartWebServer() (WebInfo, error) {
	a.webMu.Lock()
	defer a.webMu.Unlock()

	if a.web != nil {
		return webInfo(a.web), nil
	}

	backend := &webBackend{app: a}
	srv, err := webui.Start(backend, "0.0.0.0:8420")
	if err != nil {
		log.Printf("web ui: port 8420 unavailable (%v), falling back to a free port", err)
		srv, err = webui.Start(backend, "0.0.0.0:0")
	}
	if err != nil {
		return WebInfo{}, fmt.Errorf("web server: %w", err)
	}

	a.web = srv
	log.Printf("web ui: listening on %s, urls=%v", srv.Addr(), srv.URLs())
	return webInfo(srv), nil
}

// StopWebServer shuts the local web server down. It is safe to call when the
// server is not running. Active phone connections are closed immediately.
func (a *App) StopWebServer() error {
	a.webMu.Lock()
	srv := a.web
	a.web = nil
	a.webMu.Unlock()
	if srv == nil {
		return nil
	}

	err := srv.Close()
	log.Printf("web ui: stopped, err=%v", err)
	return err
}

// WebServerStatus reports whether the local web interface is running.
func (a *App) WebServerStatus() WebInfo {
	a.webMu.Lock()
	defer a.webMu.Unlock()
	if a.web == nil {
		return WebInfo{Running: false, URLs: []string{}}
	}
	return webInfo(a.web)
}

// WebServerQRCode returns a data:image/png URL with a QR code for one of the
// addresses shown in the Web dialog. The phone can scan it without typing the
// LAN IP and port.
func (a *App) WebServerQRCode(text string) (string, error) {
	a.webMu.Lock()
	running := a.web != nil
	a.webMu.Unlock()
	if !running {
		return "", errors.New("web server is not running")
	}

	text = strings.TrimSpace(text)
	u, err := url.Parse(text)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid web URL")
	}

	png, err := webui.QRCodePNG(text, 8)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

func webInfo(s *webui.Server) WebInfo {
	return WebInfo{
		Running: true,
		Port:    s.Port(),
		URLs:    s.URLs(),
		Started: s.Started().Format(time.RFC3339),
	}
}

// webBackend adapts the Wails App to the plain HTTP/SSE web UI.
type webBackend struct {
	app *App
}

func (b *webBackend) Status() webui.Status {
	st := b.app.Status()
	return webui.Status{
		Connected:  st.Connected,
		Connecting: st.Connecting,
		Port:       st.Port,
		Sweeping:   st.Sweeping,
		Model:      st.Model,
		Battery:    st.Battery,
		Version:    ver.String(),
		Config:     st.Config,
	}
}

func (b *webBackend) Ports() ([]webui.Port, error) {
	ports, err := b.app.ListPorts()
	if err != nil {
		return nil, err
	}
	out := make([]webui.Port, 0, len(ports))
	for _, p := range ports {
		out = append(out, webui.Port{
			Name:     p.Name,
			VID:      p.VID,
			PID:      p.PID,
			Serial:   p.Serial,
			Product:  p.Product,
			IsTinySA: p.IsTinySA,
		})
	}
	return out, nil
}

func (b *webBackend) Connect(port string) error {
	_, err := b.app.Connect(port)
	return err
}

func (b *webBackend) CancelConnect() { b.app.CancelConnect() }

func (b *webBackend) Disconnect() error { return b.app.Disconnect() }

func (b *webBackend) ApplySweep(cfg tinysa.SweepConfig) error { return b.app.ApplySweep(cfg) }

func (b *webBackend) StopSweep() error { return b.app.StopSweep() }

func (b *webBackend) Frame() *tinysa.Frame { return b.app.lastFrameSnapshot() }
