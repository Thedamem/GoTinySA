// SPDX-License-Identifier: GPL-3.0-or-later

package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"tinysa_go/internal/tinysa"
)

type fakeBackend struct {
	mu sync.Mutex

	status Status
	ports  []Port
	frame  *tinysa.Frame

	portsErr   error
	connectErr error
	applyErr   error

	lastPort    string
	lastCfg     tinysa.SweepConfig
	connectN    int
	cancelN     int
	disconnectN int
	applyN      int
	stopN       int
}

func (f *fakeBackend) Status() Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeBackend) Ports() ([]Port, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ports, f.portsErr
}

func (f *fakeBackend) Connect(port string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectN++
	f.lastPort = port
	return f.connectErr
}

func (f *fakeBackend) CancelConnect() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelN++
}

func (f *fakeBackend) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnectN++
	return nil
}

func (f *fakeBackend) ApplySweep(cfg tinysa.SweepConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.applyN++
	f.lastCfg = cfg
	if f.applyErr == nil {
		f.status.Sweeping = true
	}
	return f.applyErr
}

func (f *fakeBackend) StopSweep() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopN++
	f.status.Sweeping = false
	return nil
}

func (f *fakeBackend) Frame() *tinysa.Frame {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.frame
}

func (f *fakeBackend) setFrame(frame *tinysa.Frame) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frame = frame
}

type fakeCounters struct {
	connectN    int
	cancelN     int
	disconnectN int
	applyN      int
	stopN       int
	lastPort    string
	lastCfg     tinysa.SweepConfig
}

func (f *fakeBackend) counters() fakeCounters {
	f.mu.Lock()
	defer f.mu.Unlock()
	return fakeCounters{
		connectN:    f.connectN,
		cancelN:     f.cancelN,
		disconnectN: f.disconnectN,
		applyN:      f.applyN,
		stopN:       f.stopN,
		lastPort:    f.lastPort,
		lastCfg:     f.lastCfg,
	}
}

func testServer(t *testing.T, backend *fakeBackend) (*Server, string) {
	t.Helper()
	srv, err := Start(backend, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return srv, "http://" + srv.Addr()
}

func get(t *testing.T, url string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return resp, body
}

func postJSON(t *testing.T, url string, payload string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return resp, body
}

func TestStaticAssets(t *testing.T) {
	backend := &fakeBackend{}
	_, base := testServer(t, backend)

	resp, body := get(t, base+"/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("index status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "tinySA") || !strings.Contains(string(body), "app.js") {
		t.Fatalf("unexpected index: %s", body)
	}

	resp, body = get(t, base+"/app.js")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "EventSource") {
		t.Fatalf("app.js missing: status=%d len=%d", resp.StatusCode, len(body))
	}
	resp, body = get(t, base+"/style.css")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "--accent") {
		t.Fatalf("style.css missing: status=%d len=%d", resp.StatusCode, len(body))
	}
}

func TestStatusPortsFrame(t *testing.T) {
	frame := &tinysa.Frame{Seq: 7, Start: 88e6, Stop: 108e6, Points: 2, DBM: []float32{-90, -40}}
	cfg := tinysa.SweepConfig{Start: 88e6, Stop: 108e6, Points: 400, RBW: 0, Atten: -1}
	backend := &fakeBackend{
		status: Status{
			Connected: true,
			Port:      "COM7",
			Sweeping:  true,
			Model:     tinysa.Model{Ultra: true, MaxFreq: tinysa.MaxFreqUltra},
			Battery:   "4.1 V",
			Version:   "test",
			Config:    &cfg,
		},
		ports: []Port{{Name: "COM7", IsTinySA: true}},
		frame: frame,
	}
	_, base := testServer(t, backend)

	resp, body := get(t, base+"/api/status")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d", resp.StatusCode)
	}
	var st Status
	if err := json.Unmarshal(body, &st); err != nil {
		t.Fatal(err)
	}
	if !st.Connected || st.Port != "COM7" || !st.Model.Ultra {
		t.Fatalf("unexpected status: %+v", st)
	}
	if st.Config == nil || st.Config.Start != 88e6 || st.Config.Points != 400 {
		t.Fatalf("status missing sweep config: %+v", st.Config)
	}

	_, body = get(t, base+"/api/ports")
	var ports struct {
		Ports []Port `json:"ports"`
	}
	if err := json.Unmarshal(body, &ports); err != nil {
		t.Fatal(err)
	}
	if len(ports.Ports) != 1 || ports.Ports[0].Name != "COM7" {
		t.Fatalf("unexpected ports: %+v", ports.Ports)
	}

	_, body = get(t, base+"/api/frame")
	var got tinysa.Frame
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Seq != 7 || len(got.DBM) != 2 {
		t.Fatalf("unexpected frame: %+v", got)
	}
}

func TestSweepAPI(t *testing.T) {
	backend := &fakeBackend{
		status: Status{
			Connected: true,
			Model:     tinysa.Model{Ultra: true, MaxFreq: tinysa.MaxFreqUltra},
		},
	}
	_, base := testServer(t, backend)

	valid := `{"start":100000000,"stop":110000000,"points":400,"rbw":0,"atten":-1,"lna":false,"spur":false}`
	resp, body := postJSON(t, base+"/api/sweep", valid)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid sweep status=%d body=%s", resp.StatusCode, body)
	}
	if c := backend.counters(); c.applyN != 1 || c.lastCfg.Start != 100e6 {
		t.Fatalf("apply not called: %+v", c)
	}

	bad := `{"start":110000000,"stop":100000000,"points":400}`
	resp, _ = postJSON(t, base+"/api/sweep", bad)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reversed range status = %d, want 400", resp.StatusCode)
	}
	if c := backend.counters(); c.applyN != 1 {
		t.Fatalf("invalid config reached backend: %d", c.applyN)
	}

	backend.mu.Lock()
	backend.status.Connected = false
	backend.mu.Unlock()
	resp, _ = postJSON(t, base+"/api/sweep", valid)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("disconnected status = %d, want 409", resp.StatusCode)
	}

	resp, _ = postJSON(t, base+"/api/sweep/stop", "")
	if c := backend.counters(); resp.StatusCode != http.StatusOK || c.stopN != 1 {
		t.Fatalf("stop status=%d calls=%d", resp.StatusCode, c.stopN)
	}
}

func TestConnectAPI(t *testing.T) {
	backend := &fakeBackend{}
	_, base := testServer(t, backend)

	resp, _ := postJSON(t, base+"/api/connect", `{"port":"COM9"}`)
	if c := backend.counters(); resp.StatusCode != http.StatusOK || c.lastPort != "COM9" {
		t.Fatalf("connect status=%d port=%q", resp.StatusCode, c.lastPort)
	}
	resp, _ = postJSON(t, base+"/api/connect", `{"port":"  "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty port status = %d", resp.StatusCode)
	}
	resp, _ = postJSON(t, base+"/api/connect/cancel", "")
	if c := backend.counters(); resp.StatusCode != http.StatusOK || c.cancelN != 1 {
		t.Fatalf("cancel status=%d calls=%d", resp.StatusCode, c.cancelN)
	}
	resp, _ = postJSON(t, base+"/api/disconnect", "")
	if c := backend.counters(); resp.StatusCode != http.StatusOK || c.disconnectN != 1 {
		t.Fatalf("disconnect status=%d calls=%d", resp.StatusCode, c.disconnectN)
	}
}

func TestCSVAPI(t *testing.T) {
	backend := &fakeBackend{}
	_, base := testServer(t, backend)

	resp, _ := get(t, base+"/api/csv")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("empty CSV status = %d, want 409", resp.StatusCode)
	}

	backend.setFrame(&tinysa.Frame{
		Start:  100e6,
		Stop:   200e6,
		Points: 2,
		DBM:    []float32{-100, -40},
		Max:    []float32{-90, -30},
		Avg:    []float32{-95, -35},
		Min:    []float32{-110, -50},
		Noise:  -100,
	})
	resp, body := get(t, base+"/api/csv")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("CSV status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("content type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "tinysa_sweep_") {
		t.Fatalf("content disposition = %q", cd)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) != 3 {
		t.Fatalf("CSV lines = %d: %q", len(lines), body)
	}
	if !strings.HasPrefix(lines[0], "Frequency_MHz,") || !strings.HasPrefix(lines[1], "100.000000,") {
		t.Fatalf("unexpected CSV: %q", body)
	}
}

func TestEventsStream(t *testing.T) {
	backend := &fakeBackend{status: Status{Connected: true, Port: "COM3"}}
	_, base := testServer(t, backend)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("events status = %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var gotStatus, gotFrame bool
	for !gotFrame {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read event (status=%v frame=%v): %v", gotStatus, gotFrame, err)
		}
		line = strings.TrimSpace(line)
		switch line {
		case "event: status":
			if !gotStatus {
				gotStatus = true
				backend.setFrame(&tinysa.Frame{
					Seq:    2,
					Start:  1e6,
					Stop:   2e6,
					Points: 2,
					DBM:    []float32{-90, -40},
				})
			}
		case "event: frame":
			gotFrame = true
		}
	}
	if !gotStatus {
		t.Fatal("no status event")
	}
}

func readSSEEvent(t *testing.T, r *bufio.Reader) (string, []byte) {
	t.Helper()
	var name string
	var data []byte
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE event: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if name != "" {
				return name, data
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			name = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			data = []byte(strings.TrimPrefix(line, "data: "))
		}
	}
}

func TestEventsStreamSendsConfigChange(t *testing.T) {
	backend := &fakeBackend{status: Status{Connected: true}}
	_, base := testServer(t, backend)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	name, _ := readSSEEvent(t, reader)
	if name != "status" {
		t.Fatalf("first event = %q, want status", name)
	}

	cfg := tinysa.SweepConfig{Start: 100e6, Stop: 110e6, Points: 400}
	backend.mu.Lock()
	backend.status.Sweeping = true
	backend.status.Config = &cfg
	backend.mu.Unlock()

	for {
		name, data := readSSEEvent(t, reader)
		if name != "status" {
			continue
		}
		var st Status
		if err := json.Unmarshal(data, &st); err != nil {
			t.Fatal(err)
		}
		if st.Config == nil {
			continue
		}
		if !st.Sweeping || st.Config.Start != 100e6 {
			t.Fatalf("unexpected status config: %+v", st.Config)
		}
		return
	}
}

func TestQRCodePNG(t *testing.T) {
	png, err := QRCodePNG("http://192.0.2.10:8420", 8)
	if err != nil {
		t.Fatalf("QRCodePNG: %v", err)
	}
	if len(png) < 100 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not a PNG: len=%d", len(png))
	}
	if _, err := QRCodePNG("", 8); err == nil {
		t.Fatal("empty text accepted")
	}
	if _, err := QRCodePNG(strings.Repeat("x", 2000), 8); err == nil {
		t.Fatal("oversized text accepted")
	}
}

func TestLANURLs(t *testing.T) {
	urls := LANURLs(8420)
	loopback := fmt.Sprintf("http://127.0.0.1:%d", 8420)
	if len(urls) == 0 || urls[len(urls)-1] != loopback {
		t.Fatalf("loopback must be last: %v", urls)
	}
	seen := make(map[string]bool)
	for _, u := range urls {
		if seen[u] {
			t.Fatalf("duplicate URL %q in %v", u, urls)
		}
		seen[u] = true
	}
}
