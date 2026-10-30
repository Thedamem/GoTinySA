// SPDX-License-Identifier: GPL-3.0-or-later

package usblog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	textPath, binPath := l.Paths()

	l.Record(TX, []byte("version\r"))
	l.Record(RX, []byte("tinySA4_v1.4\r\nch> "))
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	recs, err := ReadBinary(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].Dir != TX || string(recs[0].Payload) != "version\r" {
		t.Fatalf("record 0 = %c %q", recs[0].Dir, recs[0].Payload)
	}
	if recs[1].Dir != RX || string(recs[1].Payload) != "tinySA4_v1.4\r\nch> " {
		t.Fatalf("record 1 = %c %q", recs[1].Dir, recs[1].Payload)
	}
	if recs[1].ElapsedUS < recs[0].ElapsedUS {
		t.Fatalf("elapsed not monotonic: %d then %d", recs[0].ElapsedUS, recs[1].ElapsedUS)
	}

	txt, err := os.ReadFile(textPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(txt, []byte("|version.|")) {
		t.Fatalf("text log does not contain ASCII dump:\n%s", txt)
	}
	if !strings.Contains(string(txt), " T 0008:") {
		t.Fatalf("text log does not contain TX header:\n%s", txt)
	}
}

func TestRotationKeepsAtMostMaxFiles(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir, Options{MaxSize: 1, MaxFiles: 3})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		l.Record(TX, []byte("x"))
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	logs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	bins, err := filepath.Glob(filepath.Join(dir, "*.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) > 3 || len(bins) > 3 {
		t.Fatalf("rotation kept too many files: %d logs, %d bins", len(logs), len(bins))
	}
	if len(logs) == 0 || len(bins) == 0 {
		t.Fatalf("no files after rotation: %d logs, %d bins", len(logs), len(bins))
	}
}

func TestNilLoggerIsNoOp(t *testing.T) {
	var l *Logger
	l.Record(TX, []byte("data")) // must not panic
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
}
