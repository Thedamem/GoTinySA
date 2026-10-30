// SPDX-License-Identifier: GPL-3.0-or-later

package serial

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tinysa_go/internal/usblog"
)

// fakePort behaves like a serial port: reads come from a scripted buffer,
// writes are captured, an empty buffer behaves like a read timeout.
type fakePort struct {
	mu        sync.Mutex
	rx        bytes.Buffer
	tx        bytes.Buffer
	to        time.Duration
	autoAbort bool
}

func (f *fakePort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rx.Len() == 0 {
		return 0, nil
	}
	return f.rx.Read(p)
}

func (f *fakePort) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err := f.tx.Write(p)
	if f.autoAbort && bytes.Contains(p, []byte("abort")) {
		f.rx.WriteString("abort\r\nch> ")
	}
	return n, err
}

func (f *fakePort) Close() error { return nil }

func (f *fakePort) SetReadTimeout(d time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.to = d
	return nil
}

func (f *fakePort) ResetInputBuffer() error { return nil }

func (f *fakePort) feed(s string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rx.WriteString(s)
}

func (f *fakePort) written() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tx.String()
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestQueryVersion(t *testing.T) {
	fp := &fakePort{}
	fp.feed("version\r\ntinySA4_v1.4-252-g2b4aa51\r\nch> ")
	c := NewClient("fake", fp, nil)
	defer c.Close()

	got, err := c.Query(testCtx(t), "version")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tinySA4_v1.4-252-g2b4aa51" {
		t.Fatalf("got %q", got)
	}
	if fp.written() != "version\r" {
		t.Fatalf("device received %q", fp.written())
	}
}

func TestQueryMultiline(t *testing.T) {
	fp := &fakePort{}
	fp.feed("info\r\ntinySA4\r\nVersion: 1.4\r\nch> ")
	c := NewClient("fake", fp, nil)
	defer c.Close()

	got, err := c.Query(testCtx(t), "info")
	if err != nil {
		t.Fatal(err)
	}
	want := "tinySA4\r\nVersion: 1.4"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestQueryTimeout(t *testing.T) {
	fp := &fakePort{} // nothing fed
	c := NewClient("fake", fp, nil)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := c.Query(ctx, "version")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got err %v, want DeadlineExceeded", err)
	}
}

func TestWriteSwallowsEcho(t *testing.T) {
	fp := &fakePort{}
	fp.feed("abort\r\nch> ")
	c := NewClient("fake", fp, nil)
	defer c.Close()

	if err := c.Write(testCtx(t), "abort"); err != nil {
		t.Fatal(err)
	}
	if fp.written() != "abort\r" {
		t.Fatalf("device received %q", fp.written())
	}

	// A following query must not see stale bytes.
	fp.feed("version\r\ntinySA4\r\nch> ")
	got, err := c.Query(testCtx(t), "version")
	if err != nil || got != "tinySA4" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestQuerySkipsStaleData(t *testing.T) {
	fp := &fakePort{}
	// Simulate leftover sweep bytes before the echo (previous session).
	fp.feed("\x00\x7f}{garbage{data}version\r\ntinySA4_v1.4-252\r\nch> ")
	c := NewClient("fake", fp, nil)
	defer c.Close()

	got, err := c.Query(testCtx(t), "version")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tinySA4_v1.4-252" {
		t.Fatalf("got %q, want clean version", got)
	}
}

func TestAbortStopsStream(t *testing.T) {
	fp := &fakePort{autoAbort: true}
	c := NewClient("fake", fp, nil)
	defer c.Close()

	if err := c.Abort(); err != nil {
		t.Fatal(err)
	}
	if tx := fp.written(); !strings.Contains(tx, "abort\r") {
		t.Fatalf("device received %q", tx)
	}
}

func TestStreamDeliversRawChunks(t *testing.T) {
	fp := &fakePort{autoAbort: true}
	fp.feed("scanraw 1 2 3 3\r\n{abc")
	c := NewClient("fake", fp, nil)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var got []byte
	done := make(chan error, 1)
	go func() {
		done <- c.Stream(ctx, "scanraw 1 2 3 3", func(b []byte) error {
			mu.Lock()
			got = append(got, b...)
			if len(got) >= 4 {
				cancel()
			}
			mu.Unlock()
			return nil
		})
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("stream error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not stop")
	}

	mu.Lock()
	data := string(got)
	mu.Unlock()
	if !strings.Contains(data, "{abc") {
		t.Fatalf("stream data = %q", data)
	}
	tx := fp.written()
	if !strings.Contains(tx, "scanraw 1 2 3 3\r") || !strings.Contains(tx, "abort\r") {
		t.Fatalf("device received %q", tx)
	}
}

func TestQueryIsLogged(t *testing.T) {
	fp := &fakePort{}
	fp.feed("version\r\ntinySA4\r\nch> ")
	log, err := usblog.New(t.TempDir(), usblog.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	_, binPath := log.Paths()

	c := NewClient("fake", fp, log)
	defer c.Close()

	if _, err := c.Query(testCtx(t), "version"); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	recs, err := usblog.ReadBinary(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].Dir != usblog.TX || string(recs[0].Payload) != "version\r" {
		t.Fatalf("record 0 = %c %q", recs[0].Dir, recs[0].Payload)
	}
	if recs[1].Dir != usblog.RX || !bytes.Contains(recs[1].Payload, []byte("tinySA4")) {
		t.Fatalf("record 1 = %c %q", recs[1].Dir, recs[1].Payload)
	}
}
