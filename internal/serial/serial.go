// SPDX-License-Identifier: GPL-3.0-or-later

// Package serial owns tinySA USB serial ports: discovery, a single-owner
// command pipeline, prompt-framed queries and raw traffic logging hooks.
//
// The device speaks a CR-terminated shell: every command is echoed back and
// every response ends with the prompt "ch> " (see docs/PROTOCOL.md).
package serial

import (
	"fmt"
	"io"
	"sort"
	"time"

	sio "go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// DefaultBaud matches QtTinySA's choice for the CDC-ACM link.
const DefaultBaud = 576000

// Prompt terminates every response from the tinySA shell.
const Prompt = "ch> "

// USB ids of the tinySA CDC interface.
const (
	TinySAVID = "0483"
	TinySAPID = "5740"
)

// Port is the subset of go.bug.st/serial.Port we rely on (kept small so tests
// can substitute a fake).
type Port interface {
	io.ReadWriteCloser
	SetReadTimeout(time.Duration) error
	ResetInputBuffer() error
}

// Open opens a real serial port. baud == 0 means DefaultBaud. CDC-ACM links
// ignore the rate, so if the driver rejects 576000 (some Windows drivers do)
// we retry with standard rates. A short retry also covers Windows releasing
// a just-closed port.
func Open(name string, baud int) (Port, error) {
	if baud == 0 {
		baud = DefaultBaud
	}
	rates := []int{baud}
	for _, fallback := range []int{115200, 9600} {
		if fallback != baud {
			rates = append(rates, fallback)
		}
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		for _, rate := range rates {
			p, err := sio.Open(name, &sio.Mode{BaudRate: rate})
			if err == nil {
				assertLines(p)
				return p, nil
			}
			lastErr = err
		}
		time.Sleep(150 * time.Millisecond)
	}
	return nil, fmt.Errorf("serial: open %s: %w", name, lastErr)
}

// OpenAt opens at exactly the given rate (used for a Windows retry).
func OpenAt(name string, baud int) (Port, error) {
	p, err := sio.Open(name, &sio.Mode{BaudRate: baud})
	if err != nil {
		return nil, fmt.Errorf("serial: open %s at %d: %w", name, baud, err)
	}
	assertLines(p)
	return p, nil
}

// assertLines raises DTR/RTS explicitly: some Windows drivers keep the CDC
// data path closed until the modem lines are asserted.
func assertLines(p Port) {
	if sp, ok := p.(interface{ SetDTR(bool) error }); ok {
		_ = sp.SetDTR(true)
	}
	if sp, ok := p.(interface{ SetRTS(bool) error }); ok {
		_ = sp.SetRTS(true)
	}
}

// Device is one discovered serial port.
type Device struct {
	Name     string // /dev/ttyACM0, COM7, ...
	VID      string // "0483" for tinySA (may be empty)
	PID      string // "5740" (may be empty)
	Serial   string
	Product  string
	IsTinySA bool
}

// FindTinySA lists ports matching the tinySA USB ids.
func FindTinySA() ([]Device, error) {
	all, err := ListPorts()
	if err != nil {
		return nil, err
	}
	var out []Device
	for _, d := range all {
		if d.IsTinySA {
			out = append(out, d)
		}
	}
	return out, nil
}

// ListPorts returns every discovered serial port. The detailed enumeration is
// best-effort: on Windows it can abort as a whole because of one broken device,
// so the plain port list is always merged in.
func ListPorts() ([]Device, error) {
	var out []Device
	seen := make(map[string]bool)

	detailed, derr := enumerator.GetDetailedPortsList()
	for _, p := range detailed {
		d := Device{
			Name:    p.Name,
			VID:     p.VID,
			PID:     p.PID,
			Serial:  p.SerialNumber,
			Product: p.Product,
		}
		d.IsTinySA = p.IsUSB && d.VID == TinySAVID && d.PID == TinySAPID
		out = append(out, d)
		seen[p.Name] = true
	}

	names, nerr := sio.GetPortsList()
	for _, name := range names {
		if !seen[name] {
			out = append(out, Device{Name: name})
		}
	}

	if len(out) == 0 {
		if derr != nil {
			return nil, fmt.Errorf("serial: enumerate: %w", derr)
		}
		if nerr != nil {
			return nil, fmt.Errorf("serial: enumerate: %w", nerr)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
