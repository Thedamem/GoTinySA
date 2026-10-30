// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"tinysa_go/internal/serial"
	"tinysa_go/internal/tinysa"
	"tinysa_go/internal/usblog"
	ver "tinysa_go/internal/version"
	"tinysa_go/internal/webui"
)

// queryTimeout bounds every one-shot console command. Sweep streaming (M2)
// will use its own long-lived path instead of Query.
const queryTimeout = 5 * time.Second

// App is the Wails-bound application object. Every exported method runs on a
// goroutine provided by the Wails runtime, never on the UI thread, so blocking
// here does not freeze the window.
type App struct {
	ctx context.Context

	mu      sync.Mutex
	client  *serial.Client
	port    string
	model   tinysa.Model
	battery string
	log     *usblog.Logger
	logDir  string

	frameMu   sync.Mutex
	lastFrame *tinysa.Frame

	connecting bool
	connectID  uint64
	pending    *serial.Client

	sweepMu  sync.Mutex
	sweep    *sweepSession
	sweepCfg *tinysa.SweepConfig

	webMu sync.Mutex
	web   *webui.Server
}

// sweepSession identifies one running sweep. The token lets a finishing old
// sweep clear only its own state, so a quick stop/start cannot lose the new
// cancellation handle.
type sweepSession struct {
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func (s *sweepSession) finish() {
	s.cancel()
	s.once.Do(func() { close(s.done) })
}

// NewApp creates the application object.
func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// beforeClose releases the port and the raw log before the window goes away.
// Cleanup is capped so a stuck driver can never prevent the window from closing.
func (a *App) beforeClose(ctx context.Context) bool {
	done := make(chan struct{})
	go func() {
		_ = a.StopSweep()
		_ = a.StopWebServer()
		_ = a.Disconnect()
		_ = a.DisableUSBLog()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1500 * time.Millisecond):
		log.Printf("beforeClose: cleanup timed out, exiting anyway")
	}
	return false
}

// Version returns the build version string.
func (a *App) Version() string { return ver.String() }

// LogPath returns the diagnostics log location ("" when logging is disabled).
func (a *App) LogPath() string { return logFilePath }

// PortInfo describes one serial port for the UI.
type PortInfo struct {
	Name     string `json:"name"`
	VID      string `json:"vid"`
	PID      string `json:"pid"`
	Serial   string `json:"serial"`
	Product  string `json:"product"`
	IsTinySA bool   `json:"isTinySA"`
}

// Status is the current connection state.
type Status struct {
	Connected  bool                `json:"connected"`
	Connecting bool                `json:"connecting"`
	Port       string              `json:"port"`
	USBDir     string              `json:"usbDir"`
	Sweeping   bool                `json:"sweeping"`
	Model      tinysa.Model        `json:"model"`
	Battery    string              `json:"battery"`
	Config     *tinysa.SweepConfig `json:"config,omitempty"`
}

// ListPorts returns every serial port known to the OS.
func (a *App) ListPorts() ([]PortInfo, error) {
	devs, err := serial.ListPorts()
	if err != nil {
		return nil, err
	}
	out := make([]PortInfo, 0, len(devs))
	for _, d := range devs {
		out = append(out, PortInfo{
			Name:     d.Name,
			VID:      d.VID,
			PID:      d.PID,
			Serial:   d.Serial,
			Product:  d.Product,
			IsTinySA: d.IsTinySA,
		})
	}
	return out, nil
}

type probeResult struct {
	version string
	client  *serial.Client
	battery string
	err     error
}

// Connect opens port, verifies the link with `version` and keeps the client.
// The whole attempt has a hard 12 s budget; the mutex is never held across I/O
// and CancelConnect can abort an attempt that is stuck in a driver call.
func (a *App) Connect(port string) (string, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return "", errors.New("port is empty")
	}

	a.mu.Lock()
	if a.client != nil {
		p := a.port
		a.mu.Unlock()
		return "", fmt.Errorf("already connected to %s", p)
	}
	if a.connecting {
		a.mu.Unlock()
		return "", errors.New("connection already in progress")
	}
	a.connecting = true
	a.connectID++
	id := a.connectID
	a.mu.Unlock()

	a.emitStep("preparing")

	results := make(chan probeResult, 1)
	go func() {
		var last probeResult
		for _, baud := range []int{serial.DefaultBaud, 115200} {
			last = a.probe(id, port, baud)
			if last.err == nil {
				results <- last
				return
			}
			a.emitStep(fmt.Sprintf("%d baud: %v", baud, last.err))
		}
		results <- last
	}()

	var res probeResult
	select {
	case res = <-results:
	case <-time.After(12 * time.Second):
		a.cancelPending(id)
		go func() {
			r := <-results
			if r.client != nil {
				_ = r.client.Close()
			}
		}()
		a.emitStep("timeout 12 s")
		return "", errors.New("connect timeout (12 s): check the cable and driver, see the log for details")
	}

	if res.err != nil {
		a.mu.Lock()
		if a.connectID == id {
			a.connecting = false
		}
		a.mu.Unlock()
		return "", fmt.Errorf("failed to connect to %s: %w (close other programs that use the port)", port, res.err)
	}

	a.mu.Lock()
	if a.connectID != id {
		a.mu.Unlock()
		_ = res.client.Close()
		return "", errors.New("connection cancelled")
	}
	a.client = res.client
	a.port = port
	a.model = tinysa.Detect(res.version)
	a.battery = res.battery
	a.pending = nil
	a.connecting = false
	a.mu.Unlock()

	a.emitStep("connected: " + res.version)
	log.Printf("connect %s: ok, %q", port, res.version)
	return res.version, nil
}

// CancelConnect aborts an in-flight connection attempt.
func (a *App) CancelConnect() {
	a.mu.Lock()
	a.connectID++
	a.connecting = false
	p := a.pending
	a.pending = nil
	a.mu.Unlock()
	if p != nil {
		log.Printf("connect: cancelled by user")
		_ = p.Close()
	}
	a.emitStep("cancelled")
}

// probe opens the port at exactly baud, stops leftover output and checks the
// device with `version`. The client is returned only on success.
func (a *App) probe(id uint64, port string, baud int) probeResult {
	a.emitStep(fmt.Sprintf("opening %s @ %d", port, baud))
	p, err := serial.OpenAt(port, baud)
	if err != nil {
		return probeResult{err: err}
	}
	client := serial.NewClient(port, p, a.log)

	a.mu.Lock()
	if a.connectID != id {
		a.mu.Unlock()
		_ = client.Close()
		return probeResult{err: errors.New("cancelled")}
	}
	a.pending = client
	a.mu.Unlock()

	clearPending := func() {
		a.mu.Lock()
		if a.pending == client {
			a.pending = nil
		}
		a.mu.Unlock()
	}

	time.Sleep(150 * time.Millisecond) // let the CDC interface settle

	// A failed abort is not fatal: some drivers lose the first write after
	// open, and the version query below may still work.
	a.emitStep("abort")
	if err := client.Abort(); err != nil {
		log.Printf("probe %s @%d: abort: %v (continuing)", port, baud, err)
		a.emitStep("abort: " + err.Error())
	}

	var version string
	var qerr error
	for attempt := 0; attempt < 2; attempt++ {
		a.emitStep(fmt.Sprintf("version (%d/2)", attempt+1))
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		version, qerr = client.Query(ctx, "version")
		cancel()
		if qerr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if qerr != nil {
		clearPending()
		_ = client.Close()
		return probeResult{err: fmt.Errorf("version: %w", qerr)}
	}

	battery := ""
	vbCtx, vbCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if vbat, err := client.Query(vbCtx, "vbat"); err == nil {
		battery = parseVBat(vbat)
	}
	vbCancel()
	clearPending()
	return probeResult{version: version, client: client, battery: battery}
}

func (a *App) cancelPending(id uint64) {
	a.mu.Lock()
	if a.connectID == id {
		a.connectID++
		a.connecting = false
	}
	p := a.pending
	a.pending = nil
	a.mu.Unlock()
	if p != nil {
		_ = p.Close()
	}
}

func (a *App) emitStep(msg string) {
	log.Printf("connect: %s", msg)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "connect:step", msg)
	}
}

// parseVBat turns the vbat response (millivolts, e.g. "4100") into "4.1 V".
func parseVBat(s string) string {
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	switch {
	case len(digits) >= 4:
		return string(digits[:1]) + "." + string(digits[1:2]) + " V"
	case len(digits) == 3:
		return string(digits[:1]) + "." + string(digits[1:3]) + " V"
	case len(digits) == 2:
		return string(digits[:1]) + "." + string(digits[1:2]) + " V"
	default:
		return ""
	}
}

// Disconnect stops any running sweep and closes the port. The raw USB log
// stays enabled for the next connection.
func (a *App) Disconnect() error {
	_ = a.StopSweep()

	a.mu.Lock()
	client := a.client
	pending := a.pending
	a.pending = nil
	a.client = nil
	a.port = ""
	a.model = tinysa.Model{}
	a.battery = ""
	a.connecting = false
	a.connectID++ // invalidate any in-flight connect
	a.mu.Unlock()

	a.sweepMu.Lock()
	a.sweepCfg = nil
	a.sweepMu.Unlock()

	if pending != nil {
		_ = pending.Close()
	}
	if client != nil {
		log.Printf("disconnect: closing %s", client.Name())
		err := client.Close()
		runtime.EventsEmit(a.ctx, "connect:step", "disconnected")
		return err
	}
	return nil
}

// Status reports the connection, sweep and raw-log state.
func (a *App) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sweepMu.Lock()
	sweeping := a.sweep != nil
	var cfg *tinysa.SweepConfig
	if a.sweepCfg != nil {
		c := *a.sweepCfg
		cfg = &c
	}
	a.sweepMu.Unlock()
	return Status{
		Connected:  a.client != nil,
		Connecting: a.connecting,
		Port:       a.port,
		USBDir:     a.logDir,
		Sweeping:   sweeping,
		Model:      a.model,
		Battery:    a.battery,
		Config:     cfg,
	}
}

// StartSweep applies cfg to the device and starts continuous scanraw
// streaming. Frames are emitted as the "sweep" event; errors as "sweep:error".
func (a *App) StartSweep(cfg tinysa.SweepConfig) error {
	a.mu.Lock()
	client := a.client
	model := a.model
	a.mu.Unlock()
	if client == nil {
		return errors.New("not connected")
	}
	if err := cfg.Validate(model); err != nil {
		return err
	}

	a.sweepMu.Lock()
	if a.sweep != nil {
		a.sweepMu.Unlock()
		return errors.New("sweep is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	sess := &sweepSession{cancel: cancel, done: make(chan struct{})}
	a.sweep = sess
	a.sweepMu.Unlock()

	sweeper := tinysa.NewSweeper(client, model)
	applyCtx, applyCancel := context.WithTimeout(ctx, 5*time.Second)
	err := sweeper.Apply(applyCtx, cfg)
	applyCancel()
	if err != nil {
		a.finishSweep(sess)
		return err
	}

	cfgCopy := cfg
	a.sweepMu.Lock()
	a.sweepCfg = &cfgCopy
	a.sweepMu.Unlock()

	runtime.EventsEmit(a.ctx, "sweep:started", cfg)
	go func() {
		defer a.finishSweep(sess)
		var last time.Time
		err := sweeper.Run(ctx, cfg, func(f tinysa.Frame) {
			a.frameMu.Lock()
			a.lastFrame = &f
			a.frameMu.Unlock()
			if time.Since(last) < 60*time.Millisecond { // ~16 UI updates/s
				return
			}
			last = time.Now()
			runtime.EventsEmit(a.ctx, "sweep", f)
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			runtime.EventsEmit(a.ctx, "sweep:error", err.Error())
		}
		runtime.EventsEmit(a.ctx, "sweep:stopped", nil)
	}()
	return nil
}

// ApplySweep validates cfg and (re)starts the sweep. A running sweep is
// stopped and awaited first, so the web UI can change the range with a single
// request without racing the previous stream.
func (a *App) ApplySweep(cfg tinysa.SweepConfig) error {
	a.mu.Lock()
	client := a.client
	model := a.model
	a.mu.Unlock()
	if client == nil {
		return errors.New("not connected")
	}
	if err := cfg.Validate(model); err != nil {
		return err
	}

	a.sweepMu.Lock()
	old := a.sweep
	a.sweepMu.Unlock()
	if old != nil {
		old.cancel()
		select {
		case <-old.done:
		case <-time.After(3 * time.Second):
			return errors.New("current sweep did not stop")
		}
	}
	return a.StartSweep(cfg)
}

// StopSweep cancels the running sweep; the stream sends abort and stops.
func (a *App) StopSweep() error {
	a.sweepMu.Lock()
	sess := a.sweep
	a.sweepMu.Unlock()
	if sess != nil {
		sess.cancel()
	}
	return nil
}

func (a *App) finishSweep(sess *sweepSession) {
	sess.cancel()
	a.sweepMu.Lock()
	if a.sweep == sess {
		a.sweep = nil
	}
	a.sweepMu.Unlock()
	sess.finish()
}

// Query sends one shell command and returns the response text.
func (a *App) Query(cmd string) (string, error) {
	cmd = strings.TrimSpace(strings.TrimSuffix(cmd, "\r"))
	if cmd == "" {
		return "", errors.New("empty command")
	}

	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil {
		return "", errors.New("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	return client.Query(ctx, cmd)
}

// SaveCSV asks for a destination and writes the latest sweep with all traces.
func (a *App) SaveCSV() (string, error) {
	a.frameMu.Lock()
	f := a.lastFrame
	a.frameMu.Unlock()
	if f == nil {
		return "", errors.New("no sweep data yet")
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save sweep as CSV",
		DefaultFilename: "tinysa_sweep.csv",
		Filters:         []runtime.FileFilter{{DisplayName: "CSV files", Pattern: "*.csv"}},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // cancelled
	}

	if err := os.WriteFile(path, f.CSV(), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// lastFrameSnapshot returns the latest completed sweep without copying its
// slices. Frames are immutable once published, so the web server may encode
// them while a newer frame is being assembled.
func (a *App) lastFrameSnapshot() *tinysa.Frame {
	a.frameMu.Lock()
	defer a.frameMu.Unlock()
	return a.lastFrame
}

// EnableUSBLog starts raw traffic logging into dir (text + binary, see
// internal/usblog). Must be called while disconnected. Returns both paths.
func (a *App) EnableUSBLog(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", errors.New("directory is empty")
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.connecting {
		return "", errors.New("connection in progress")
	}
	if a.client != nil {
		return "", errors.New("disconnect before enabling the USB log")
	}
	if a.log != nil {
		return "", errors.New("USB log is already enabled")
	}

	l, err := usblog.New(dir, usblog.DefaultOptions())
	if err != nil {
		return "", err
	}
	a.log = l
	a.logDir = dir
	tp, bp := l.Paths()
	return tp + "\n" + bp, nil
}

// DisableUSBLog stops and closes raw logging. Must be called while
// disconnected.
func (a *App) DisableUSBLog() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.connecting {
		return errors.New("connection in progress")
	}
	if a.client != nil {
		return errors.New("disconnect before disabling the USB log")
	}
	if a.log == nil {
		return nil
	}
	err := a.log.Close()
	a.log = nil
	a.logDir = ""
	return err
}
